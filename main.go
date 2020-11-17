package main

import (
	"encoding/json"
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
		Use:    "init",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheName := args[0]
			if argConfigInit.kind == "" || argConfigInit.container == "" || len(argConfigInit.fingerprints) == 0 {
				log.Fatal().Msg("kind, fingerprint, and container must all be specified")
			}

			// use existing signing key, or make a new one
			var privateKeyStr string
			var publicKeyStr string
			var err error
			if privateKeyStr, found := os.LookupEnv("NICHE_SIGNING_KEY"); found {
				pubFromPrivateKeyStr := func(s string) (string, error) {
					return "", nil
				}
				publicKeyStr, err = pubFromPrivateKeyStr(privateKeyStr)
				if err != nil {
					return err
				}
				log.Info().Str("publicKey", publicKeyStr).Msgf("Using signingkey from ${NICHE_SIGNING_KEY}")
			} else {
				log.Info().Msgf("Generating new signing key")
				if privateKeyStr, publicKeyStr, err = nixStoreGenerateBinaryCacheKey(cacheName); err != nil {
					return err
				}
				log.Info().Str("publicKey", publicKeyStr).Msgf("Generated new signing key")
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

			log.Info().Msg("successfully created new niche repo")
			// TODO: get Stow to have a NiceURL() that we can use here?

			return nil
		},
	}
	cmdConfigInit.PersistentFlags().StringVarP(&argConfigInit.kind, "kind", "k", "", "the 'kind' of storage to use (from graymeta/stow)")
	cmdConfigInit.PersistentFlags().StringVarP(&argConfigInit.container, "container", "c", "", "the name of the container to use (aws bucket, azure container name, etc)")
	cmdConfigInit.PersistentFlags().StringSliceVarP(&argConfigInit.fingerprints, "fingerprints", "p", []string{}, "the gpg fingerprint(s) to use for encrypting/decrypting the config (list multiple times, and/or comma separated)")
	cmdConfig.AddCommand(cmdConfigInit)

	var argConfigDownload struct {
		configFilePath string
	}
	var cmdConfigDownload = &cobra.Command{
		Use:  "download",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheURLStr := args[0]
			cacheURL, err := preprocessHostArg(cacheURLStr)
			if err != nil {
				return err
			}

			if argConfigDownload.configFilePath == "" {
				log.Fatal().Msg("'config' argument is required")
			}

			c, err := clientFromSops(*cacheURL)
			if err != nil {
				return err
			}
			defer c.stowClient.Close()

			data, err := json.MarshalIndent(c.config, "", "  ")
			if err != nil {
				return err
			}
			ioutil.WriteFile(argConfigDownload.configFilePath, data, 0600)

			log.Info().Str("configFile", argConfigDownload.configFilePath).Msg("config download complete")

			return nil
		},
	}
	cmdConfigDownload.PersistentFlags().StringVarP(&argConfigDownload.configFilePath, "config", "f", "", "where to save the downloaded and decrypted config")
	cmdConfig.AddCommand(cmdConfigDownload)

	var argConfigUpload struct {
		configFilePath string
	}
	var cmdConfigUpload = &cobra.Command{
		Use: "upload",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromFile(argConfigUpload.configFilePath)
			if err != nil {
				return err
			}
			defer c.stowClient.Close()

			// TODO: (inside reupload config) generate public config json? uploadConfigs() does both?

			err = c.reuploadConfig() // TODO: rename func?
			if err != nil {
				return err
			}

			log.Info().Str("configFile", argConfigUpload.configFilePath).Msg("config upload complete")

			return nil
		},
	}
	cmdConfigUpload.PersistentFlags().StringVarP(&argConfigUpload.configFilePath, "config", "f", "", "path to config file to init/force overwrite")
	cmdConfig.AddCommand(cmdConfigUpload)

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

	rootCmd.Execute()
}
