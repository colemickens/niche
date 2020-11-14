package main

import (
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
	_ "github.com/graymeta/stow/sftp"
	_ "github.com/graymeta/stow/swift"
)

func preprocessHost(host string) (*url.URL, error) {
	if !strings.HasPrefix(host, "https://") || !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	return url.Parse(host)
}

func processBuildQueue(c *nicheClient, queue chan string, wg *sync.WaitGroup, alwaysOverwrite bool) {
	wg.Add(1)
	defer wg.Done()

	seenPaths := []string{}

	for storePath := range queue {
		if storePath == "QUIT" {
			log.Println("leaving build queue")
			return
		}
		for _, seenPath := range seenPaths {
			if strings.EqualFold(storePath, seenPath) {
				continue
			}
		}
		c.ensurePath(storePath, alwaysOverwrite)
		seenPaths = append(seenPaths, storePath)
		log.Println("ensured", storePath)
	}
}

func main() {
	var rootCmd = &cobra.Command{Use: "niche"}

	var argQueue struct {
		socketPath string
	}
	var cmdQueue = &cobra.Command{
		Use:    "queue",
		Hidden: true,
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

	var argReconfigure struct {
		cache          string
		configFilePath string
	}
	var cmdReconfigure = &cobra.Command{
		Use: "reconfigure",
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheURL, err := preprocessHost(argReconfigure.cache)
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
				// TODO: does it make sense to deserialize and then reserialize?
				//  -   we could just hold onto the raw bytes when we load+parse?
				oldBytes, err := json.Marshal(c.config)
				if err != nil {
					return err
				}
				newConfigBytes, err := CaptureInputFromEditor(oldBytes)
				if err != nil {
					return err
				}
				// update client with a new one built from the new config
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
	cmdReconfigure.PersistentFlags().StringVarP(&argReconfigure.cache, "cache-url", "u", "", "cache url")
	cmdReconfigure.PersistentFlags().StringVarP(&argReconfigure.configFilePath, "config-file", "c", "", "path to config file to init/force overwrite")
	rootCmd.AddCommand(cmdReconfigure)

	var argBuild struct {
		cache string
	}
	var cmdBuild = &cobra.Command{
		Use:   "build",
		Short: "builds an INSTALLABLE and uploads each output as they're built",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			extraArgs := args
			_ = extraArgs // TODO: Fix this

			cacheURL, err := preprocessHost(argBuild.cache)
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
			go listen(c, socketPath, queue)
			go processBuildQueue(c, queue, &wg, alwaysOverwrite)

			err = build(*cacheURL, socketPath, extraArgs...)
			if err != nil {
				return err
			}

			wg.Wait()

			log.Println("all done")
			return nil
		},
	}
	cmdBuild.PersistentFlags().StringVarP(&argBuild.cache, "cache-url", "u", "", "cache url")
	rootCmd.AddCommand(cmdBuild)

	rootCmd.Execute()
}
