package main

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/colemickens/niche/pkg/nixclient"
)

var nix nixclient.NixClient = nixclient.NixClientCli{}

func preprocessHostArg(host string) (*url.URL, error) {
	if !strings.HasPrefix(host, "https://") && !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	return url.Parse(host)
}

//
// NICHE QUEUE
var argQueue struct {
	socketPath string
}

func getCmdQueue() *cobra.Command {
	cmdQueue := &cobra.Command{
		Use:    "queue",
		Hidden: true, // only used internally, in all usages
		RunE: func(cmd *cobra.Command, args []string) error {
			return queue(argQueue.socketPath)
		},
	}
	cmdQueue.PersistentFlags().StringVarP(&argQueue.socketPath, "socket", "s", "", "path of the socket to write paths to")
	return cmdQueue
}

//
// NICHE CONFIG INIT
var argConfigInit struct {
	kind         string
	container    string
	fingerprints []string
}

func getCmdConfigInit() *cobra.Command {
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
	return cmdConfigInit
}

//
// NICHE CONFIG DOWNLOAD
var argConfigDownload struct {
	configFilePath string
}

func getCmdConfigDownload() *cobra.Command {
	cmdConfigDownload := &cobra.Command{
		Use:  "download",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configDownload(args[0], argConfigDownload.configFilePath)
		},
	}
	cmdConfigDownload.PersistentFlags().StringVarP(&argConfigDownload.configFilePath, "config", "f", "", "where to save the downloaded and decrypted config")
	return cmdConfigDownload
}

//
// NICHE CONFIG UPLOAD
var argConfigUpload struct {
	configFilePath string
}

func getCmdConfigUpload() *cobra.Command {
	cmdConfigUpload := &cobra.Command{
		Use: "upload",
		RunE: func(cmd *cobra.Command, args []string) error {
			return configUpload(argConfigUpload.configFilePath)
		},
	}
	cmdConfigUpload.PersistentFlags().StringVarP(&argConfigUpload.configFilePath, "config", "f", "", "path to config file to init/force overwrite")
	return cmdConfigUpload
}

//
// NICHE SHOW
func getCmdShow() *cobra.Command {
	cmdShow := &cobra.Command{
		Use: "show",
		RunE: func(cmd *cobra.Command, args []string) error {
			return show(args[0])
		},
	}
	return cmdShow
}

//
// NICHE BUILD
func getCmdBuild() *cobra.Command {
	cmdBuild := &cobra.Command{
		Use:   "build",
		Short: "builds an INSTALLABLE and uploads each output as they're built",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return build(args[0], args[1:])
		},
	}
	return cmdBuild
}

func mainCobra() error {
	var rootCmd = &cobra.Command{Use: "niche"}

	rootCmd.AddCommand(getCmdQueue())

	cmdConfig := &cobra.Command{
		Use:   "config",
		Short: "commands to download/upload/initialize a config file",
	}
	cmdConfig.AddCommand(getCmdConfigInit())
	cmdConfig.AddCommand(getCmdConfigDownload())
	cmdConfig.AddCommand(getCmdConfigUpload())
	rootCmd.AddCommand(cmdConfig)

	rootCmd.AddCommand(getCmdShow())

	rootCmd.AddCommand(getCmdBuild())

	// TODO: rootCmd.AddCommand(cmdUpload)

	return rootCmd.Execute()
}
