package niche

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type pathHandler func(path string) error

type processor struct {
	client       *nicheClient
	alwaysUpload bool

	wg     *sync.WaitGroup
	queue  <-chan string
	quit   chan bool
	exited chan bool
}

func newProcessor(queue chan string, client *nicheClient, alwaysUpload bool) *processor {
	return &processor{
		client:       client,
		alwaysUpload: alwaysUpload,
		wg:           &sync.WaitGroup{},
		queue:        queue,
		quit:         make(chan bool),
		exited:       make(chan bool),
	}
}

func (p *processor) stop() {
	close(p.quit)
	<-p.exited
}

// process dedupes among the paths it has seen itself
func (p *processor) process() {
	p.wg.Add(1)
	defer func() {
		p.wg.Done()
		log.Trace().Msg("leaving build queue")

		log.Trace().Msg("closing build queue")
		p.exited <- true
	}()

	seenPaths := []string{}

outer:
	for {
		select {
		case <-p.quit:
			return
		case storePath := <-p.queue:
			if storePath == "QUIT" {
				return
			}
			// if we've seen this path, skip it, continue the otuer loop
			for _, seenPath := range seenPaths {
				if strings.EqualFold(storePath, seenPath) {
					continue outer
				}
			}

			allStorePaths, err := p.client.nix.QueryPaths(storePath)
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

				var err error
				if p.alwaysUpload {
					err = p.client.uploadPath(storePath)
				} else {
					err = p.client.ensurePath(storePath)
				}
				if err != nil {
					log.Warn().Err(err).Msg("handler failed")
				}
				seenPaths = append(seenPaths, storePath)
			}

		default:
		}
	}
}
