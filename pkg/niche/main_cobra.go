package niche

import (
	"github.com/spf13/cobra"
)

func mainCobra() error {
	//
	// NICHE QUEUE
	var argQueue struct {
		socketPath string
	}
	cmdQueue := &cobra.Command{
		Use:    "queue",
		Hidden: true, // only used internally, in all usages
		RunE: func(cmd *cobra.Command, args []string) error {
			return queue(argQueue.socketPath)
		},
	}
	cmdQueue.PersistentFlags().StringVarP(&argQueue.socketPath, "socket", "s", "", "path of the socket to write paths to")

	//
	// NICHE CONFIG INIT
	var argConfigInit struct {
		kind         string
		container    string
		fingerprints []string
	}

	cmdConfigInit := &cobra.Command{
		Use:    "init",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configInit(args[0], argConfigInit.kind, argConfigInit.container, argConfigInit.fingerprints)
		},
	}
	cmdConfigInit.PersistentFlags().StringVarP(&argConfigInit.kind, "kind", "k", "", "the 'kind' of storage to use (from graymeta/stow)")
	cmdConfigInit.PersistentFlags().StringVarP(&argConfigInit.container, "container", "c", "", "the name of the container to use (aws bucket, azure container name, etc)")
	cmdConfigInit.PersistentFlags().StringSliceVarP(&argConfigInit.fingerprints, "fingerprints", "p", []string{}, "the gpg fingerprint(s) to use for encrypting/decrypting the config (list multiple times, and/or comma separated)")

	//
	// NICHE CONFIG DOWNLOAD
	var argConfigDownload struct {
		configFilePath string
	}

	cmdConfigDownload := &cobra.Command{
		Use:  "download",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configDownload(args[0], argConfigDownload.configFilePath)
		},
	}
	cmdConfigDownload.PersistentFlags().StringVarP(&argConfigDownload.configFilePath, "config", "f", "", "where to save the downloaded and decrypted config")

	//
	// NICHE CONFIG UPLOAD
	var argConfigUpload struct {
		configFilePath string
	}

	cmdConfigUpload := &cobra.Command{
		Use: "upload",
		RunE: func(cmd *cobra.Command, args []string) error {
			return configUpload(argConfigUpload.configFilePath)
		},
	}
	cmdConfigUpload.PersistentFlags().StringVarP(&argConfigUpload.configFilePath, "config", "f", "", "path to config file to init/force overwrite")

	//
	// NICHE SHOW
	cmdShow := &cobra.Command{
		Use: "show",
		RunE: func(cmd *cobra.Command, args []string) error {
			return show(args[0])
		},
	}

	//
	// NICHE BUILD
	cmdBuild := &cobra.Command{
		Use:   "build",
		Short: "builds an INSTALLABLE and uploads each output as they're built",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return build(args[0], args[1:], false)
		},
	}

	var rootCmd = &cobra.Command{Use: "niche"}

	rootCmd.AddCommand(cmdQueue)

	cmdConfig := &cobra.Command{
		Use:   "config",
		Short: "commands to download/upload/initialize a config file",
	}
	cmdConfig.AddCommand(cmdConfigInit)
	cmdConfig.AddCommand(cmdConfigDownload)
	cmdConfig.AddCommand(cmdConfigUpload)
	rootCmd.AddCommand(cmdConfig)

	rootCmd.AddCommand(cmdShow)

	rootCmd.AddCommand(cmdBuild)

	// TODO: rootCmd.AddCommand(cmdUpload)

	return rootCmd.Execute()
}
