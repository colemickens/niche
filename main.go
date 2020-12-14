package main

import (
	"fmt"
	"os"

	"github.com/colemickens/niche/pkg/niche"

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
	log.Info().Str("logLevel", level).Msg("zerolog configured")
}

func main() {
	err := niche.MainCli()
	if err != nil {
		fmt.Printf("mainCli returned error, exit 1 (err=%s)\n", err)
		os.Exit(1)
	}
}
