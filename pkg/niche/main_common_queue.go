package niche

import (
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func queue(socketPath string) error {
	outPathsStr := os.Getenv("OUT_PATHS")
	outPaths := strings.Split(outPathsStr, " ")

	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	defer c.Close()

	for _, p := range outPaths {
		_, err = c.Write([]byte(p + "\n"))
		if err != nil {
			return err
		}
		log.Trace().Str("storePath", p).Msg("sent path to socket")
	}

	return nil
}
