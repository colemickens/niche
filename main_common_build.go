package main

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
)

const numUploaders int = 1

func build(cacheNameRaw string, extraArgs []string, alwaysOverwrite bool) error {
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

	queue := make(chan string, (1<<16)-1)

	listener, err := newReceiver(c.nix, socketPath, queue)
	if err != nil {
		return err
	}
	go listener.run()
	defer listener.close()

	// process the build queue
	wg := sync.WaitGroup{}
	for i := 0; i < numUploaders; i++ {
		go processUploadQueue(c, queue, &wg, alwaysOverwrite)
	}
	defer wg.Wait()

	outLink, err := nix.Build(socketPath, extraArgs...)
	if err != nil {
		return err
	}

	finalBuiltPaths, err := nix.QueryPaths(outLink)
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	for _, p := range finalBuiltPaths {
		_, err = conn.Write([]byte(p + "\n"))
		if err != nil {
			return err
		}
		log.Info().Str("path", p).Msg("sent final out link")
	}

	err = os.RemoveAll(outLink)
	if err != nil {
		return err
	}
	log.Info().Str("outLink", outLink).Msg("cleaned up outlink")
	_, err = conn.Write([]byte("QUIT\n"))
	if err != nil {
		return err
	}
	log.Info().Msg("sent QUIT")
	return nil
}
