package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz"
)

type nixClient interface {
	PathInfo(storePath, narItemPath string) (narInfo *narInfo, err error)
	HashFile(storePath string)
	DumpPath(storePath string)
	ToBase32(storePath string) (hash string, err error)
	QueryPaths(storePath string) (dependencies []string, err error)
	Build(thing string)
}

type nixCliClient struct{}

// this is actually narInfo for storePath, and specific nar.......
// we might just take filepath to nar instead of hash+Size
func (_ nixCliClient) PathInfo(storePath, narItemPath string) (nar *narInfo, err error) {
	fileHash := "TODO"
	fileSize := int64(0)

	pathInfoCmd := exec.Command("nix", "path-info", "--json", storePath)
	pathInfoBytes, err := pathInfoCmd.Output()
	if err != nil {
		return nil, err
	}

	var info []struct {
		Path       string `json:"path"`
		NarHash    string
		NarSize    int64
		References []string
		Deriver    string
		Signatures []string
	}
	s := string(pathInfoBytes)
	_ = s
	err = json.Unmarshal(pathInfoBytes, &info)
	if err != nil {
		return nil, err
	}

	return &narInfo{
		URL:         narItemPath,
		Compression: "xz", // TODO: derive from narItemPath suffix
		StorePath:   info[0].Path,
		FileHash:    fileHash,
		FileSize:    fileSize,
		NarHash:     info[0].NarHash,
		NarSize:     info[0].NarSize,
		References:  info[0].References,
		Deriver:     filepath.Base(info[0].Deriver),
		Signatures:  info[0].Signatures,
	}, nil
}

func (_ nixCliClient) HashFile(storePath string) (hash string, err error) {
	hashCmd := exec.Command("nix", "hash-file", storePath)
	hashBytes, err := hashCmd.Output()
	fileHash := strings.TrimSpace(string(hashBytes))
	return fileHash, nil
}

func (_ nixCliClient) DumpPath(storePath string) (string, error) {
	// compress + upload the NAR
	tempFilePath := filepath.Join(os.TempDir(), "nix-dump-path.tmp") // TODO: add random
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return "", err
	}

	xzWriter, err := xz.NewWriter(tempFile)
	if err != nil {
		log.Fatal().Err(err).Msg("xz.NewWriter error")
		return "", err
	}

	dumpCmd := exec.Command("nix", "dump-path", storePath)

	dumpCmdStdout, err := dumpCmd.StdoutPipe()
	if err != nil {
		log.Fatal().Err(err).Msg("dumpCmd StdoutPipe error")
		return "", err
	}

	err = dumpCmd.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("dumpCmd Start error")
		return "", err
	}

	var outwg sync.WaitGroup
	outwg.Add(1)
	go func() {
		n, err := io.Copy(xzWriter, dumpCmdStdout)
		if err != nil {
			log.Fatal().Err(err).Msg("error copying the xz stream to file")
		}
		log.Info().Int64("bytesCopied", n).Msg("copied bytes to file from xz stream")
		xzWriter.Close()
		tempFile.Close()
		outwg.Done()
	}()
	outwg.Wait()

	if err := dumpCmd.Wait(); err != nil {
		log.Fatal().Err(err).Msg("dumpCmd: failed to wait")
	}

	log.Info().Str("storePath", storePath).Msg("done waiting for process/copy")
	return tempFilePath, nil
}

func (_ nixCliClient) ToBase32(hash string) (string, error) {
	cmd := exec.Command("nix", "to-base32", hash)
	hashStrBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(hashStrBytes)), nil
}

func (_ nixCliClient) QueryStorePaths(storePath string) ([]string, error) {
	// nix-store -q -R $storePath
	cmd := exec.Command("nix-store", "-q", "-R", storePath)
	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	splitFn := func(c rune) bool {
		return (c == ' ' || c == '\n' || c == '\r')
	}
	outputStrings := strings.FieldsFunc(string(outputBytes), splitFn)

	return outputStrings, nil
}

func (_ nixCliClient) Build(cacheURL url.URL, socketPath string, buildArgs ...string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	postBuildBody := fmt.Sprintf("#!/bin/sh\n%s queue -s %s", self, socketPath)
	postBuildHookPath := "/tmp/pbh"
	f, err := os.Create(postBuildHookPath)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(postBuildHookPath, []byte(postBuildBody), 0777)
	if err != nil {
		return err
	}
	f.Close()
	if err := os.Chmod(postBuildHookPath, 0777); err != nil {
		return err
	}

	outLink := "/tmp/outlink" // TODO TODO TODO TODO
	// TODO rm outlink just in case
	nbArgs := []string{"build"}
	nbArgs = append(nbArgs, buildArgs...)
	nbArgs = append(nbArgs, "--option", "post-build-hook", postBuildHookPath, "--out-link", outLink)
	cmd := exec.Command("nix", nbArgs...)

	_, err = cmd.Output()
	if eerr, ok := err.(*exec.ExitError); ok {
		log.Info().Msg(string(eerr.Stderr))
		panic(eerr)
	}

	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	defer c.Close()
	_, err = c.Write([]byte(outLink + "\n"))
	if err != nil {
		return err
	}
	log.Info().Str("outLink", outLink).Msg("sent final out link")
	_, err = c.Write([]byte("QUIT\n"))
	if err != nil {
		return err
	}
	log.Info().Msg("sent QUIT")
	return nil
}
