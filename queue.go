package main

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// TODO: eventually this could be a struct too, with a Run()
// and the caller could easily spawn multiple processors,
// if we wanted to be able to handle multiple paths concurrently

func processBuildQueue(c *nicheClient, queue chan string, wg *sync.WaitGroup, alwaysOverwrite bool) {
	wg.Add(1)
	defer wg.Done()

	seenPaths := []string{}

	for storePath := range queue {
		if storePath == "QUIT" {
			log.Info().Msg("leaving build queue")
			return
		}
		for _, seenPath := range seenPaths {
			if strings.EqualFold(storePath, seenPath) {
				log.Trace().Str("storePath", storePath).Msg("skipping already processed path")
				continue
			}
		}
		if alwaysOverwrite {
			c.uploadPath(storePath)
			log.Info().Str("storePath", storePath).Msg("uploaded storePath")
		} else {
			c.ensurePath(storePath)
			log.Info().Str("storePath", storePath).Msg("ensured storePath")
		}
		seenPaths = append(seenPaths, storePath)
	}
}
