package main

import (
	"bufio"
	"net"
	"os"
	"strings"

	"github.com/colemickens/niche/pkg/nixclient"
	"github.com/rs/zerolog/log"
)

type receiver struct {
	listener net.Listener
	queue    chan<- string
	nix      nixclient.NixClient
}

func newReceiver(nix nixclient.NixClient, socketPath string, q chan<- string) (receiver, error) {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Err(err).Str("path", socketPath).Msg("failed to clean up socket ahead of time")
	}
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Err(err).Msg("failed to listen")
	}
	r := receiver{
		listener: l,
		queue:    q,
		nix:      nix,
	}
	return r, nil
}

func (r *receiver) close() error {
	return r.listener.Close()
}

func (r *receiver) run() error {
	for {
		// either accept a new connection, or accept an error back from... enqueuing?
		conn, err := r.listener.Accept()
		if err != nil {
			log.Err(err).Msg("failed to accept new connection")
			break
		}

		log.Trace().
			Str("remoteAddr", conn.RemoteAddr().String()).
			Msg("accepted new connection")

		go handle(r.nix, conn, r.queue)
	}
	log.Trace().Msg("closing listener")
	r.listener.Close()
	return nil
}

func handle(nix nixclient.NixClient, conn net.Conn, queue chan<- string) {
	defer func() {
		log.Info().Str("remoteAddr", conn.RemoteAddr().String()).Msg("closing connection")
		conn.Close()
	}()

	bufReader := bufio.NewReader(conn)
	for {
		byts, err := bufReader.ReadBytes('\n')
		if err != nil {
			log.Warn().Err(err).Msg("?????")
			break
		}
		storePath := strings.TrimSpace(string(byts))
		log.Trace().Str("storePath", storePath).Msg("received storePath")

		if storePath == "QUIT" {
			log.Trace().Msg("told to quit")
			queue <- storePath
			break
		}

		// our "socket" accepts all sorts of paths
		// we throw them at nix-store and try to get back a list of paths
		// that's what we actually queue for handling as individal good store paths
		allStorePaths, err := nix.QueryPaths(storePath)
		if err != nil {
			log.Warn().Err(err).Msg("?????")
			break
		}

		for _, storePath := range allStorePaths {
			log.Info().Str("storePath", storePath).Msg("sending storePath to the queue")
			queue <- storePath
		}
	}
}
