package main

import (
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
)

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
			cacheURL, err := url.Parse(args[0])
			if err != nil {
				return err
			}

			var newConfig unversionedNicheConfig
			if argReconfigure.configFilePath != "" {
				newConfig, err = configFromFilePath(argReconfigure.configFilePath)
				if err != nil {
					return err
				}
			} else {
				newConfig = unversionedNicheConfig{}
			}

			// TODO: sanity check, warn if keys change?
			err = replaceConfig(*cacheURL, newConfig)
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
			cacheURL, err := url.Parse(args[0])
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

				go listen(socketPath, *cacheURL) // TODO: quit channel
				if err != nil {
					return err
				}
			}

			err = build(*cacheURL, socketPath)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmdBuild.PersistentFlags().StringVarP(&argBuild.cache, "cache-url", "u", "", "cache url")
	rootCmd.AddCommand(cmdBuild)

	rootCmd.Execute()
}
