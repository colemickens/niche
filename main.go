package main

import (
	"fmt"
	"os"

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
	if os.Getenv("NICHE_DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}

func main() {
	err := mainCli()
	if err != nil {
		fmt.Println("niche handling this one", err)
		os.Exit(1)
	}
}
