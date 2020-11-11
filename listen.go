package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"
)

func echoServer(conn net.Conn) {
	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
	}()

	timeoutDuration := 1000 * time.Second // TODO?
	bufReader := bufio.NewReader(conn)

	for {
		//conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		bytes, err := bufReader.ReadBytes('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%s", bytes)
		storePath := string(bytes)

		allStorePaths, err := getAllStorePaths(storePath)
		if err != nil {
			return err
		}

		for path := range allStorePaths {
			// check against our substituters
			// if not, compress, make narinfo, upload both with stow
			err := signAndCompressAndUploadStorePath(key, storePath)
			if err != nil {
				return err
			}
		}
	}
}

func listen(socketPath string, cacheURL url.URL) error {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	for {
		// Accept new connections, dispatching them to echoServer
		// in a goroutine.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}

		go echoServer(conn)
	}
	defer os.RemoveAll(socketPath)
	return nil
}
