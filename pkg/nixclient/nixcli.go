package nixclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type NixClientCli struct{}

var _ NixClient = NixClientCli{}
var _ NixClient = (*NixClientCli)(nil)

// this is actually narInfo for storePath, and specific nar.......
// we might just take filepath to nar instead of hash+Size
func (NixClientCli) PathInfo(storePath string) (pathInfo *NixPathInfo, err error) {
	pathInfoCmd := exec.Command("nix", "path-info", "--json", storePath)
	pathInfoBytes, err := pathInfoCmd.Output()
	if err != nil {
		return nil, err
	}

	var info []NixPathInfo
	s := string(pathInfoBytes)
	_ = s
	err = json.Unmarshal(pathInfoBytes, &info)
	if err != nil {
		return nil, err
	}

	return &NixPathInfo{
		Path:       storePath,
		NarHash:    info[0].NarHash,
		NarSize:    info[0].NarSize,
		References: info[0].References,
		Deriver:    filepath.Base(info[0].Deriver),
		Signatures: info[0].Signatures,
	}, nil
}

/*
func (NixClientCli) ToBase32(hash string) (string, error) {
	cmd := exec.Command("nix", "to-base32", hash)
	hashStrBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(hashStrBytes)), nil
}*/

func (NixClientCli) QueryPaths(storePath string) ([]string, error) {
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

func (nixc NixClientCli) Build(socketPath string, buildArgs ...string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}

	// unset LD_PRELOAD to work around this:
	postBuildBody := fmt.Sprintf("#!/bin/sh\n"+"unset LD_PRELOAD\n"+"%s queue -s %s", self, socketPath)

	postBuildHookPath := fmt.Sprintf("/tmp/niche_%d_pbh", os.Getpid())
	f, err := os.Create(postBuildHookPath)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(postBuildHookPath, []byte(postBuildBody), 0777)
	if err != nil {
		return "", err
	}
	f.Close()
	if err := os.Chmod(postBuildHookPath, 0777); err != nil {
		return "", err
	}
	defer func() {
		//  os.Unlink(f) // TODO: unlink PBH to get rid of it
	}()

	outLink := fmt.Sprintf("/tmp/niche_%d_outlink", os.Getpid())
	nbArgs := []string{"build"}
	nbArgs = append(nbArgs, buildArgs...)
	nbArgs = append(nbArgs, "--option", "post-build-hook", postBuildHookPath, "--out-link", outLink)
	fullCmd := append([]string{"nix"}, nbArgs...)
	cmd := exec.Command(fullCmd[0], fullCmd[1:]...)

	log.Info().Strs("cmd", fullCmd).Msg("calling nix")

	// TODO output/error handling
	_, err = cmd.Output()
	if eerr, ok := err.(*exec.ExitError); ok {
		log.Warn().Msg(string(eerr.Stderr))
		return "", err
	}

	return outLink, nil
}
