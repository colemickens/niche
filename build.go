package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// TODO: somewhere a Key is going to get passed around
// consider a struct that we can stash stuff in and call methods on

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

	err = cmd.Run()
	if err != nil {
		return err
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
