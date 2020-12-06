package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/colemickens/niche/pkg/nixclient"
	_ "github.com/graymeta/stow/azure"
	_ "github.com/graymeta/stow/b2"
	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/oracle"
	_ "github.com/graymeta/stow/s3"
	_ "github.com/graymeta/stow/swift"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
        nicheDebugEnv := os.Getenv("NICHE_DEBUG")
        level := "unset"
	if nicheDebugEnv != "" {
                level = "trace"
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else {
                level = "info"
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
        }
        log.Info().Str("logLevel", level).Msg("logging configured")
}

var nix nixclient.NixClient = nixclient.NixClientCli{}

func preprocessHostArg(host string) (*url.URL, error) {
	if host == "" {
		return nil, fmt.Errorf("niche-url (-u) must be specified")
	}
	if !strings.HasPrefix(host, "https://") && !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	return url.Parse(host)
}

func main() {
	err := mainCli()
	if err != nil {
		fmt.Println("niche handling this one", err)
		os.Exit(1)
	}
}
