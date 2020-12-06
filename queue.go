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

	for storePath := range queue {
		if storePath == "QUIT" {
			log.Info().Msg("leaving build queue")
			return
		}
		for _, seenPath := range seenPaths {
			if strings.EqualFold(storePath, seenPath) {
				continue
			}
		}
		if alwaysOverwrite {
			c.uploadPath(storePath)
		} else {
			c.ensurePath(storePath)
		}
		seenPaths = append(seenPaths, storePath)
	}
}
