package main

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// TODO: eventually this could be a struct too, with a Run()
// and the caller could easily spawn multiple processors,
// if we wanted to be able to handle multiple paths concurrently

func processUploadQueue(c *nicheClient, queue chan string, wg *sync.WaitGroup, alwaysOverwrite bool) {
	wg.Add(1)
	defer wg.Done()

	seenPaths := []string{}

outer:
	for storePath := range queue {
		// if we've seen this path, skip it
		if storePath == "QUIT" {
			log.Info().Msg("leaving build queue")
			return
		}
		// if we've seen this path, skip it, continue the otuer loop
		for _, seenPath := range seenPaths {
			if strings.EqualFold(storePath, seenPath) {
				continue outer
			}
		}

		allStorePaths, err := nix.QueryPaths(storePath)
		if err != nil {
			log.Warn().Err(err).Msg("unexpected error querying all store paths")
			break
		}

	inner:
		for _, innerStorePath := range allStorePaths {
			// if we've seen this path, skip it, continue to the next "innerStorePath"
			for _, seenPath := range seenPaths {
				if strings.EqualFold(innerStorePath, seenPath) {
					continue inner
				}
			}
			if alwaysOverwrite {
				c.uploadPath(innerStorePath)
			} else {
				c.ensurePath(innerStorePath)
			}
			seenPaths = append(seenPaths, storePath)
		}
	}
}
