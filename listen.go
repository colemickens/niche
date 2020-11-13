package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func handle(c *nicheClient, conn net.Conn, alwaysOverwrite bool, exit chan error) {
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
			exit <- err
			return
		}
		storePath := strings.Trim(string(byts), " \r\n")

		if storePath == "QUIT" {
			break
		}

		allStorePaths, err := getAllStorePaths(storePath)
		if err != nil {
			exit <- err
			return
		}

		for _, storePath := range allStorePaths {
			// check against our substituters
			// if not, compress, make narinfo, upload both with stow
			err = c.ensurePath(storePath, alwaysOverwrite)
			if err != nil {
				exit <- err
				return
			}
		}
		fmt.Println("handled all paths")
	}
	fmt.Printf("Quitting the listen loop for %s\n", conn.RemoteAddr())
}

// TODO: we need to write to a single queue
// right now each build client get its own queue
// which is also what cachix does and it seems bad
func listen(c *nicheClient, socketPath string, alwaysOverwrite bool) error {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	//for {
	// Accept new connections, dispatching them to echoServer
	// in a goroutine.
	conn, err := l.Accept()
	if err != nil {
		log.Fatal("accept error:", err)
	}

	exit := make(chan error)

	handle(c, conn, alwaysOverwrite, exit)
	//}
	defer os.RemoveAll(socketPath)
	return nil
}
