package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	_ "github.com/graymeta/stow/azure"
	_ "github.com/graymeta/stow/b2"
	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/oracle"
	_ "github.com/graymeta/stow/s3"
	_ "github.com/graymeta/stow/swift"
)

func preprocessHostArg(host string) (*url.URL, error) {
	if !strings.HasPrefix(host, "https://") || !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	return url.Parse(host)
}

func main() {
	var rootCmd = &cobra.Command{Use: "niche"}

	var argQueue struct {
		socketPath string
	}
	var cmdQueue = &cobra.Command{
		Use:    "queue",
		Hidden: true, // this, for now, is only used internally as the post-build-hook from `niche build` -> `nix build`
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
			}

			return nil
		},
	}
	cmdQueue.PersistentFlags().StringVarP(&argQueue.socketPath, "socket", "s", "", "path of the socket to write paths to")
	rootCmd.AddCommand(cmdQueue)

	var argInit struct {
		kind           string
		container      string
		gpgFingerprint string
	}
	var cmdInit = &cobra.Command{
		Use:    "init",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheName := args[1]
			if argInit.kind == "" || argInit.container == "" || argInit.gpgFingerprint == "" {
				log.Fatalln("kind, fingerprint, and container must all be specified")
			}
			privateKeyStr, publicKeyStr, err := nixStoreGenerateBinaryCacheKey(cacheName)
			if err != nil {
				return err
			}
			configMap, err := getInitialStorageConfigMap(argInit.kind)
			if err != nil {
				return err
			}

			newConfig := privateNicheConfig{
				StorageKind:      argInit.kind,
				SigningKey:       privateKeyStr,
				PublicKey:        publicKeyStr,
				StorageContainer: argInit.container,
				StorageConfigMap: configMap,
				KeyGroups:        []nicheKeyGroup{{"pgp": []string{argInit.gpgFingerprint}}},
			}

			data, err := json.MarshalIndent(newConfig, "", "  ")
			if err != nil {
				return err
			}

			// TODO: do the reconfigure dance now where we let them edit the file
			// maybe keep them in a loop??? IDK

			//
			//
			//
			//
			//
			//
			//
			//

			ioutil.WriteFile("/tmp/foo", data, 0644)

			return nil
		},
	}
	cmdInit.PersistentFlags().StringVarP(&argInit.kind, "kind", "k", "", "the 'kind' of storage to use (from graymeta/stow)")
	cmdInit.PersistentFlags().StringVarP(&argInit.kind, "container", "c", "", "the name of the container to use (aws bucket, azure container name, etc)")
	cmdInit.PersistentFlags().StringVarP(&argInit.gpgFingerprint, "fingerprint", "f", "", "the gpg fingerprint(s) to use for encrypting/decrypting the config (comma separated)")
	rootCmd.AddCommand(cmdInit)

	var argReconfigure struct {
		configFilePath string
	}
	var cmdReconfigure = &cobra.Command{
		Use: "reconfigure",
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheURLStr := args[0]
			//cacheURL, err := preprocessHost(argReconfigure.cache)
			cacheURL, err := preprocessHostArg(cacheURLStr)
			if err != nil {
				return err
			}

			var c *nicheClient
			if argReconfigure.configFilePath != "" {
				c, err = clientFromFile(argReconfigure.configFilePath)
				if err != nil {
					return err
				}
			} else {
				c, err = clientFromSops(*cacheURL)
				if err != nil {
					return err
				}
				oldBytes, err := json.Marshal(c.config)
				if err != nil {
					return err
				}
				newConfigBytes, err := CaptureInputFromEditor(oldBytes)
				if err != nil {
					return err
				}
				c, err = clientFromBytes(newConfigBytes)
				if err != nil {
					return err
				}
			}
			defer c.stowClient.Close()

			// TODO: sanity check, warn if keys change?
			err = c.reuploadConfig()
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmdReconfigure.PersistentFlags().StringVarP(&argReconfigure.configFilePath, "config", "c", "", "path to config file to init/force overwrite")
	rootCmd.AddCommand(cmdReconfigure)

	// var argBuild struct {
	// 	cache string
	// }
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
			//cacheURL, err := preprocessHost(argBuild.cache)
			if err != nil {
				return err
			}

			//socketPath := argBuild.socketPath
			socketPath := ""
			if socketPath == "" {
				dir, err := ioutil.TempDir("", "niche")
				if err != nil {
					return err
				}
				defer os.RemoveAll(dir)
				socketPath = filepath.Join(dir, "queue.sock")
			}

			c, err := clientFromSops(*cacheURL)
			if err != nil {
				return nil
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

			log.Println("all done")
			return nil
		},
	}
	//cmdBuild.PersistentFlags().StringVarP(&argBuild.cache, "cache-url", "u", "", "cache url")
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
					log.Println("uhhh BAD", err)
					break
				}

				storePath := strings.TrimSpace(string(byts))
				log.Println("received", storePath)

				if err = c.ensurePath(storePath, alwaysOverwrite); err != nil {
					log.Fatalln("failed to push", storePath)
				}
			}

			return nil
		},
	}
	rootCmd.AddCommand(cmdUpload)

	rootCmd.Execute()
}
