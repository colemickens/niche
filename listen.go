package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func handle(c *nicheClient, conn net.Conn, queue chan string) {
	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
	}()

	//timeoutDuration := 1000 * time.Second // TODO?
	bufReader := bufio.NewReader(conn)

	for {
		//conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		byts, err := bufReader.ReadBytes('\n')
		if err != nil {
			log.Println("uhhh BAD", err)
			break
		}
		storePath := strings.TrimSpace(string(byts))
		log.Println("received", storePath)

		if storePath == "QUIT" {
			log.Println("told to quit")
			queue <- storePath
			break
		}

		allStorePaths, err := getAllStorePaths(storePath)
		if err != nil {
			log.Println("uhhh BAD", err)
			break
		}

		for _, storePath := range allStorePaths {
			log.Println("propagating", storePath)
			queue <- storePath
		}
	}
}

// TODO: we need to write to a single queue
// right now each build client get its own queue
// which is also what cachix does and it seems bad
func listen(c *nicheClient, socketPath string, queue chan string) error {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go handle(c, conn, queue)
	}
}
