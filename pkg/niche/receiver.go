package niche

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type receiver struct {
	socketPath string
	listener   net.Listener
	queue      chan<- string
	quit       chan bool
	exited     chan bool
}

func newReceiver(socketPath string, q chan<- string) (receiver, error) {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Err(err).Str("path", socketPath).Msg("failed to clean up socket ahead of time")
	}
	r := receiver{
		socketPath: socketPath,
		listener:   nil,
		queue:      q,
		quit:       make(chan bool),
		exited:     make(chan bool),
	}
	return r, nil
}

func (r *receiver) stop() {
	close(r.quit)
	<-r.exited
}

func (r *receiver) run() error {
	wg := sync.WaitGroup{}

	var err error
	r.listener, err = net.Listen("unix", r.socketPath)
	if err != nil {
		log.Err(err).Msg("failed to listen")
		return err
	}

	defer func() {
		wg.Wait()
		r.listener.Close()
		r.exited <- true
	}()

	for {
		select {
		case <-r.quit:
			return nil
		default:
			conn, err := r.listener.Accept()
			if err != nil {
				log.Err(err).Msg("failed to accept new connection")
				break
			}
			log.Trace().Msg("new socket connection")
			go handle(conn, r.queue, wg)
		}
	}
}

// handle intentionally doesn't take any reference to the receiver
// since this could be handling multiple threads at the same time
// it's safe to send to teh chan from multiple "threads" though
func handle(conn net.Conn, queue chan<- string, wg sync.WaitGroup) {
	defer func() {
		log.Info().Str("remoteAddr", conn.RemoteAddr().String()).Msg("closing connection")
		wg.Done()
		conn.Close()
	}()
	wg.Add(1)

	bufReader := bufio.NewReader(conn)
	for {
		byts, err := bufReader.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			// if err == EOF ignore
			log.Warn().Err(err).Msg("failed reading bytes")
		} else if errors.Is(err, io.EOF) {
			log.Trace().Err(err).Msg("EOF")
		}
		storePath := strings.TrimSpace(string(byts))
		log.Trace().Str("storePath", storePath).Msg("received storePath")

		if storePath == "QUIT" {
			log.Trace().Msg("told to quit")
			queue <- storePath
			break
		}

		log.Trace().Str("storePath", storePath).Msg("enqueued storePath")
		queue <- storePath
	}
}
