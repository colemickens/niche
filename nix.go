package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

type nixPathInfoOutput struct {
	Path       string `json:"path"`
	NarHash    string
	NarSize    int64
	References []string
	Deriver    string
	Signatures []string
}

// this is actually narInfo for storePath, and specific nar.......
// we might just take filepath to nar instead of hash+Size
func narInfoForPath(storePath, narItemPath, fileHash string, fileSize int64) (nar *narInfo, err error) {
	pathInfoCmd := exec.Command("nix", "path-info", "--json", storePath)
	pathInfoBytes, err := pathInfoCmd.Output()
	if err != nil {
		return nil, err
	}

	var info []nixPathInfoOutput
	s := string(pathInfoBytes)
	_ = s
	err = json.Unmarshal(pathInfoBytes, &info)
	if err != nil {
		return nil, err
	}

	return &narInfo{
		URL:         narItemPath,
		Compression: "xz", // TODO: function is less generic than named
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

func nixToBase32(hash string) (string, error) {
	cmd := exec.Command("nix", "to-base32", hash)
	hashStrBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(hashStrBytes)), nil
}

func getAllStorePaths(storePath string) ([]string, error) {
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

func build(cacheURL url.URL, socketPath string, buildArgs ...string) error {
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
		log.Println(string(eerr.Stderr))
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
	log.Println("sent final build:", outLink)
	_, err = c.Write([]byte("QUIT\n"))
	if err != nil {
		return err
	}
	log.Println("sent QUIT")
	return nil
}
