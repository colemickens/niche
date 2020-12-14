package niche

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

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

	// niche client
	c, err := clientFromSops(*cacheURL)
	if err != nil {
		return err
	}
	defer c.stowClient.Close()

	// upload queue
	queue := make(chan string, (1<<16)-1)

	// processor (receives from queue, passes to handler)
	p := newProcessor(queue, c, alwaysOverwrite)
	for i := 0; i < numUploaders; i++ {
		go p.process()
	}
	defer p.stop()

	// path receiver (listens on socketPath, sends to queue)
	r, err := newReceiver(socketPath, queue)
	if err != nil {
		return err
	}
	go r.run()
	defer r.stop()

	// initate build (blocks)
	outLink, err := c.nix.Build(socketPath, extraArgs...)
	if err != nil {
		return err
	}
	defer func() {
		err = os.RemoveAll(outLink)
		if err != nil {
			log.Warn().Str("outLink", outLink).Err(err).Msg("error cleaning up outlink")
		} else {
			log.Info().Str("outLink", outLink).Msg("cleaned up outlink")
		}
	}()

	// send final out link over
	// TODO: this isn't smart enough to do the right thing for derivations with multiple out paths
	// TODO:
	// this could instead look at the derivation
	// -> recursively walk the derviation inputs
	// -> upload any of the those outputs?
	// this works around PBH not executing for non-built built paths too
	finalBuiltPaths, err := c.nix.QueryPaths(outLink)
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
	}
	log.Trace().Str("outLink", outLink).Msg("sent final outLink")

	// tell the queue to QUIT, we're done
	_, err = conn.Write([]byte("QUIT\n"))
	if err != nil {
		return err
	}
	log.Trace().Msg("sent QUIT")
	return nil
}
