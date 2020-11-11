package niche

import (
	"os"

	"github.com/urfave/cli/v2"
)

func MainCli() error {
	app := &cli.App{
		Name:  "niche",
		Usage: "a tool to manage self-service, bring-your-own-storage Nix caches",
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelpAndExit(c, c.Command.FullName(), 1)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:   "queue",
				Usage:  "queue a new path to be uploaded to a niche listener",
				Hidden: true,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "socket-path",
						Aliases: []string{"s"},
						Value:   "",
						Usage:   "the path to the listening niche socket",
					},
				},
				Action: func(c *cli.Context) error {
					socketPath := c.String("socket-path")
					return queue(socketPath)
				},
			},
			{
				Name:  "config",
				Usage: "commands for working with a niche repo configuration",
				Action: func(c *cli.Context) error {
					cli.ShowSubcommandHelp(c)
					return cli.Exit("specify a subcommand", 1)
				},
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "initialize a new repo and config file",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "cache-name",
								Aliases: []string{"n"},
								Value:   "",
								Usage:   "the name of the cache",
							},
							&cli.StringFlag{
								Name:    "kind",
								Aliases: []string{"k"},
								Value:   "",
								Usage:   "the storage provider (azure,s3,google,...)",
							},
							&cli.StringFlag{
								Name:    "bucket",
								Aliases: []string{"b"},
								Value:   "",
								Usage:   "the name of the bucket/container to create",
							},
							&cli.StringSliceFlag{
								Name:    "fingerprints",
								Aliases: []string{"p"},
								Value:   cli.NewStringSlice(),
								Usage:   "the GPG fingerprints to use for encryption",
							},
						},
						Action: func(c *cli.Context) error {
							cacheNameRaw := c.String("cache-name")
							kind := c.String("kind")
							bucket := c.String("bucket")
							fingerprints := c.StringSlice("fingerprints")
							return configInit(cacheNameRaw, kind, bucket, fingerprints)
						},
					},
					{
						Name:  "download",
						Usage: "download a repo's config file and store it decrypted",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "cache-url",
								Aliases: []string{"u"},
								Value:   "",
								Usage:   "the url to the cache",
							},
							&cli.StringFlag{
								Name:    "file",
								Aliases: []string{"f"},
								Value:   "/tmp/niche-config",
								Usage:   "the path to store the decrypted config file",
							},
						},
						Action: func(c *cli.Context) error {
							cacheURLRaw := c.String("cache-url")
							path := c.String("file")
							return configDownload(cacheURLRaw, path)
						},
					},
					{
						Name:  "upload",
						Usage: "upload a repo's config file and encrypt it on-the-fly",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "file",
								Aliases: []string{"f"},
								Value:   "/tmp/niche-config",
								Usage:   "the path to encrypt and upload as the config file",
							},
						},
						Action: func(c *cli.Context) error {
							path := c.String("file")
							return configUpload(path)
						},
					},
				},
			},

			{
				Name:  "build",
				Usage: "build a nix installable and upload as paths are built",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "cache-url",
						Aliases: []string{"u"},
						Value:   "",
						Usage:   "the url to the cache",
					},
					&cli.BoolFlag{
						Name:    "always-upload",
						Aliases: []string{"a"},
						Value:   false,
						Usage:   "always/force upload, regardless of upstream cache or previous existence",
					},
				},
				Action: func(c *cli.Context) error {
					cacheURLRaw := c.String("cache-url")
					return build(cacheURLRaw, c.Args().Slice()[0:], c.Bool("always-upload"))
				},
			},

			{
				Name:  "show",
				Usage: "show the public config for a repo",
				Action: func(c *cli.Context) error {
					return show(c.Args().Get(0))
				},
			},
		},
	}

	return app.Run(os.Args)
}
