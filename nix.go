package main

import (
	"crypto/ed25519"
	"encoding/base64"
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
	"time"

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
		Deriver:     info[0].Deriver,
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

func (ni *narInfo) AddSignature(privateKeyStr string) error {
	// look for a sig with our prefix
	// if not found calculate sig, add
	parts := strings.Split(privateKeyStr, ":")
	serverID, privateKey := parts[0], parts[1]

	pkBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return err
	}

	// TODO: this is getting parsed repeatedly
	// we should do this earlier when we parse config
	// or store them split out in the config
	pk := ed25519.NewKeyFromSeed(pkBytes)

	found := false
	for _, curSig := range ni.Signatures {
		if strings.Contains(curSig, serverID) {
			found = true
			break
		}
	}

	if !found {
		fp := ni.Fingerprint()
		sig := ed25519.Sign(pk, fp)
		sigB64 := base64.StdEncoding.EncodeToString(sig)
		ni.Signatures = append(ni.Signatures, string(sigB64))
	}

	return nil
}

func (ni *narInfo) Fingerprint() []byte {
	// CACHIX
	// https://github.com/cachix/cachix/blob/master/cachix-api/src/Cachix/API/Signing.hs#L21-L26
	/*
		fingerprint storePath narHash narSize references =
			toS $
				T.intercalate
				";"
				["1", storePath, narHash, show narSize, T.intercalate "," references]
	*/

	// NIX
	// https://github.com/NixOS/nix/blob/7f56cf67bac3731ed8e217170eb548bf0fd2cfcb/src/libstore/store-api.cc#L918-L928
	/*
		return
			"1;" + store.printStorePath(path) + ";"
			+ narHash.to_string(Base32, true) + ";"
			+ std::to_string(narSize) + ";"
			+ concatStringsSep(",", store.printStorePathSet(references));
	*/

	fp := strings.Join(
		[]string{
			"1",
			ni.StorePath,
			ni.NarHash, // TODO: base32
			fmt.Sprintf("%d", ni.NarSize),
			strings.Join(ni.References, ","),
		}, ",")

	return []byte(fp)
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

	for {
		time.Sleep(time.Second * 1)
	}

	return nil
}
