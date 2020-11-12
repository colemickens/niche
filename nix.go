package main

/*
basically this entire file should be replaced with
better stuff from go-nix or go-wrapped https://github.com/andir/libnixstore-c
*/

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/numtide/go-nix/src/libstore"
	"github.com/ulikunitz/xz"
)

type NixPathInfoOutput struct {
	Path       string `json:"path"`
	NarHash    string
	NarSize    int
	References []string
	Deriver    string
	Signatures []string
}

func nixPathInfo(storePath string) (*libstore.NarInfo, error) {
	pathInfoCmd := exec.Command("nix", "path-info", storePath)
	pathInfoBytes, err := pathInfoCmd.Output()
	if err != nil {
		return nil, err
	}

	var info NixPathInfoOutput
	err = json.Unmarshal(pathInfoBytes, &info)
	if err != nil {
		return nil, err
	}

	return &libstore.NarInfo{
		StorePath:  info.Path,
		NarHash:    info.NarHash,
		NarSize:    info.NarSize,
		References: info.References,
		Deriver:    info.Deriver,
		Signatures: info.Signatures,
	}, nil
}

func generateSignatureWithKey(data, key string) (string, error) {
	return "", nil
}

func nixDumpPath(storePath string) (string, error) {
	// compress + upload the NAR
	tempFilePath := filepath.Join(os.TempDir(), "nix-dump-path.tmp") // TODO: add random
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return "", err
	}

	xzWriter, err := xz.NewWriter(tempFile)
	if err != nil {
		log.Fatalf("xz.NewWriter error %s", err)
		return "", err
	}

	dumpCmd := exec.Command("nix", "dump-path", storePath)
	dumpCmdStdout, err := dumpCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("dumpCmd.StdoutPipe error %s", err)
		return "", err
	}
	go func() {
		io.Copy(xzWriter, dumpCmdStdout)
	}()
	dumpCmd.Start()
	if err := dumpCmd.Wait(); err != nil {
		log.Fatal(err)
		return "", err
	}
	tempFile.Close()

	return tempFilePath, nil
}

func getAllStorePaths(storePath string) ([]string, error) {
	// nix-store -q -R $storePath
	cmd := exec.Command("nix-store", "-q", "-R", storePath)
	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	outputStrings := strings.Split(string(outputBytes), "\n")

	return outputStrings, nil
}

func build(cacheURL url.URL, socketPath string, buildArgs ...string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	postBuildHook := fmt.Sprintf("%s queue -s %s", self, socketPath)

	outLink := "/tmp/outlink" // TODO TODO TODO TODO
	// TODO rm outlink just in case
	nbArgs := []string{"build"}
	nbArgs = append(nbArgs, buildArgs...)
	nbArgs = append(nbArgs, "--option", "post-build-hook", postBuildHook, "--out-link", outLink)
	cmd := exec.Command("nix", nbArgs...)

	_, err = cmd.Output()
	if eerr, ok := err.(*exec.ExitError); ok {
		fmt.Println(string(eerr.Stderr))
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

	return nil
}
