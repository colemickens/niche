package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
)

func build(cacheNameRaw string, extraArgs []string) error {
	cacheURL, err := preprocessHostArg(cacheNameRaw)
	if err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "niche")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	socketPath := filepath.Join(dir, "queue.sock")

	c, err := clientFromSops(*cacheURL)
	if err != nil {
		return err
	}
	defer c.stowClient.Close()

	_, alwaysOverwrite := os.LookupEnv("NICHE_OVERWRITE")

	wg := sync.WaitGroup{}
	queue := make(chan string, 1000)

	listener, err := newReceiver(c.nix, socketPath, queue)
	if err != nil {
		return err
	}
	go listener.run()
	defer listener.close()

	// process the build queue
	go processBuildQueue(c, queue, &wg, alwaysOverwrite)

	err = nix.Build(socketPath, extraArgs...)
	if err != nil {
		return err
	}

	wg.Wait()
	log.Info().Msg("all done.")
	return nil
}
