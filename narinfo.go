package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

type narInfo struct {
	StorePath   string
	URL         string
	Compression string
	FileHash    string
	FileSize    int64
	NarHash     string
	NarSize     int64
	References  []string
	Deriver     string
	System      string
	Signatures  []string
	CA          string
}

func (ni narInfo) String() string {
	out := ""
	out += "StorePath: " + ni.StorePath + "\n"
	out += "URL: " + ni.URL + "\n"
	out += "Compression: " + ni.Compression + "\n"
	out += "FileHash: " + ni.FileHash + "\n"
	out += fmt.Sprintf("FileSize: %d\n", ni.FileSize)
	out += "NarHash: " + ni.NarHash + "\n"
	out += fmt.Sprintf("NarSize: %d\n", ni.NarSize)
	out += "References: " + strings.Join(ni.References, " ") + "\n"

	if ni.Deriver != "" {
		out += "Deriver: " + ni.Deriver + "\n"
	}

	if ni.System != "" {
		out += "System: " + ni.System + "\n"
	}

	for _, sig := range ni.Signatures {
		out += "Sig: " + sig + "\n"
	}

	if ni.CA != "" {
		out += "CA: " + ni.CA + "\n"
	}

	return out
}

// ContentType returns the mime content type of the object
func (ni narInfo) ContentType() string {
	return "text/x-nix-narinfo"
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
	pk := ed25519.PrivateKey(pkBytes)

	found := false
	for _, curSig := range ni.Signatures {
		if strings.Contains(curSig, serverID) {
			found = true
			break
		}
	}

	if !found {
		fp, err := ni.Fingerprint()
		if err != nil {
			return err
		}
		sig := ed25519.Sign(pk, fp)
		sigB64 := base64.StdEncoding.EncodeToString(sig)
		newSig := fmt.Sprintf("%s:%s", serverID, sigB64)
		ni.Signatures = append(ni.Signatures, newSig)
	}

	return nil
}

func (ni *narInfo) Fingerprint() ([]byte, error) {
	narHash, err := nixToBase32(ni.NarHash)
	if err != nil {
		return nil, err
	}
	narHashPrefixed := fmt.Sprintf("sha256:%s", narHash)

	fp := strings.Join(
		[]string{
			"1",
			ni.StorePath,
			narHashPrefixed,
			fmt.Sprintf("%d", ni.NarSize),
			strings.Join(ni.References, ","),
		}, ";")

	return []byte(fp), nil
}
