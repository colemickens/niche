package main

import (
	"bufio"
	"io/ioutil"
	"regexp"

	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"

	_ "github.com/graymeta/stow/azure"
	_ "github.com/graymeta/stow/b2"
	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/oracle"
	_ "github.com/graymeta/stow/s3"
	_ "github.com/graymeta/stow/swift"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func preprocessHostArg(host string) (*url.URL, error) {
	if !strings.HasPrefix(host, "https://") || !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	return url.Parse(host)
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("NICHE_DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}

func main() {
	var rootCmd = &cobra.Command{Use: "niche"}

	var argQueue struct {
		socketPath string
	}
	var cmdQueue = &cobra.Command{
		Use:    "queue",
		Hidden: true, // only used internally, in all usages
		RunE: func(cmd *cobra.Command, args []string) error {
			outPathsStr := os.Getenv("OUT_PATHS")
			outPaths := strings.Split(outPathsStr, " ")

			c, err := net.Dial("unix", argQueue.socketPath)
			if err != nil {
				return err
			}
			defer c.Close()

			for _, p := range outPaths {
				_, err = c.Write([]byte(p + "\n"))
				if err != nil {
					return err
				}
				log.Trace().Str("storePath", p).Msg("sent path to socket")
			}

			return nil
		},
	}
	cmdQueue.PersistentFlags().StringVarP(&argQueue.socketPath, "socket", "s", "", "path of the socket to write paths to")
	rootCmd.AddCommand(cmdQueue)

	cmdConfig := &cobra.Command{
		Use:   "config",
		Short: "commands to download/upload/initialize a config file",
	}

	var argConfigInit struct {
		kind         string
		container    string
		fingerprints []string
	}
	var cmdConfigInit = &cobra.Command{
		Use:    "config init",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheName := args[1]
			if argConfigInit.kind == "" || argConfigInit.container == "" || len(argConfigInit.fingerprints) == 0 {
				log.Fatal().Msg("kind, fingerprint, and container must all be specified")
			}

			// use existing signing key, or make a new one
			var privateKeyStr string
			var publicKeyStr string
			var err error
			if privateKeyStr, found := os.LookupEnv("NICHE_SIGNING_KEY"); found {
				log.Trace().Msgf("Using signingkey from NICHE_SIGNING_KEY env var")
				pubFromPrivateKeyStr := func(s string) (string, error) {
					return "", nil
				}
				publicKeyStr, err = pubFromPrivateKeyStr(privateKeyStr)
				if err != nil {
					return err
				}
			}
			if privateKeyStr, publicKeyStr, err = nixStoreGenerateBinaryCacheKey(cacheName); err != nil {
				return err
			}

			// create stow's config map from expected env vars
			configMap := getInitialStorageConfigMap(argConfigInit.kind)

			newConfig := privateNicheConfig{
				StorageKind:      argConfigInit.kind,
				SigningKey:       privateKeyStr,
				PublicKey:        publicKeyStr,
				StorageContainer: argConfigInit.container,
				StorageConfigMap: configMap,
				KeyGroups:        []nicheKeyGroup{{"pgp": argConfigInit.fingerprints}},
			}

			c, err := clientFromPrivateNicheConfig(newConfig, true)
			if err != nil {
				return err
			}

			err = c.reuploadConfig()
			if err != nil {
				return err
			}

			//u := "generate this URL basic on a template"
			log.Info().Msg("successfully created repo")
			log.Info().Msg("I think you need to determine your own url here...")
			log.Info().Msg("stow doesn't seem to give me a url to container and it looks like Item.URL() is useless")

			return nil
		},
	}
	cmdConfigInit.PersistentFlags().StringVarP(&argConfigInit.kind, "kind", "k", "", "the 'kind' of storage to use (from graymeta/stow)")
	cmdConfigInit.PersistentFlags().StringVarP(&argConfigInit.kind, "container", "c", "", "the name of the container to use (aws bucket, azure container name, etc)")
	cmdConfigInit.PersistentFlags().StringSliceVarP(&argConfigInit.fingerprints, "fingerprint", "f", []string{}, "the gpg fingerprint(s) to use for encrypting/decrypting the config (list multiple times, and/or comma separated)")
	cmdConfig.AddCommand(cmdConfigInit)

	var argConfigUpload struct {
		configFilePath string
	}
	var cmdConfigUpload = &cobra.Command{
		Use: "config upload",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromFile(argConfigUpload.configFilePath)
			if err != nil {
				return err
			}
			defer c.stowClient.Close()

			// TODO: generate public config json? uploadConfigs() does both?

			err = c.reuploadConfig() // TODO: rename func?
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmdConfigUpload.PersistentFlags().StringVarP(&argConfigUpload.configFilePath, "config", "f", "", "path to config file to init/force overwrite")
	cmdConfig.AddCommand(cmdConfigUpload)

	var cmdConfigDownload = &cobra.Command{
		Use: "config download",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromFile(argConfigDownload.configFilePath)
			if err != nil {
				return err
			}
			defer c.stowClient.Close()

			// TODO: generate public config json? DownloadConfigs() does both?

			err = c.reDownloadConfig() // TODO: rename func?
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmdConfig.AddCommand(cmdConfigDownload)

	rootCmd.AddCommand(cmdConfig)

	var cmdBuild = &cobra.Command{
		Use:   "build",
		Short: "builds an INSTALLABLE and uploads each output as they're built",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			cacheURLStr := args[0]
			extraArgs := []string{}
			if len(args) > 1 {
				extraArgs = args[1:]
			}

			cacheURL, err := preprocessHostArg(cacheURLStr)
			if err != nil {
				return err
			}

			dir, err := ioutil.TempDir("", "niche")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			socketPath := filepath.Join(dir, "queue.sock")

			c, err := clientFromSops(*cacheURL)
			if err != nil {
				return err
			}
			defer c.stowClient.Close()

			_, alwaysOverwrite := os.LookupEnv("NICHE_OVERWRITE")

			wg := sync.WaitGroup{}
			queue := make(chan string, 1000)

			// start accepting clients
			go c.listen(socketPath, queue)
			go c.processBuildQueue(queue, &wg, alwaysOverwrite)

			err = nixBuild(*cacheURL, socketPath, extraArgs...)
			if err != nil {
				return err
			}

			wg.Wait()

			log.Info().Msg("done")
			return nil
		},
	}
	rootCmd.AddCommand(cmdBuild)

	var cmdUpload = &cobra.Command{
		Use:   "upload",
		Short: "uploads paths piped into standard in",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			cacheURLStr := args[0]

			cacheURL, err := preprocessHostArg(cacheURLStr)
			//cacheURL, err := preprocessHost(argBuild.cache)
			if err != nil {
				return err
			}

			c, err := clientFromSops(*cacheURL)
			if err != nil {
				return nil
			}
			defer c.stowClient.Close()

			_, alwaysOverwrite := os.LookupEnv("NICHE_OVERWRITE")

			//timeoutDuration := 1000 * time.Second // TODO?
			bufReader := bufio.NewReader(os.Stdin)

			for {
				byts, err := bufReader.ReadBytes('\n')
				if err != nil {
					log.Warn().Err(err).Msg("uhhh BAD")
					break
				}

				storePath := strings.TrimSpace(string(byts))
				if err = c.ensurePath(storePath, alwaysOverwrite); err != nil {
					log.Warn().Err(err).Msgf("failed to upload %s", storePath)
				}
			}

			return nil
		},
	}
	rootCmd.AddCommand(cmdUpload)

	rootCmd.Execute()
}
