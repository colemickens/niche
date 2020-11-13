package main

import (
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

func (n narInfo) String() string {
	out := ""
	out += "StorePath: " + n.StorePath + "\n"
	out += "URL: " + n.URL + "\n"
	out += "Compression: " + n.Compression + "\n"
	out += "FileHash: " + n.FileHash + "\n"
	out += fmt.Sprintf("FileSize: %d\n", n.FileSize)
	out += "NarHash: " + n.NarHash + "\n"
	out += fmt.Sprintf("NarSize: %d\n", n.NarSize)
	out += "References: " + strings.Join(n.References, " ") + "\n"

	if n.Deriver != "" {
		out += "Deriver: " + n.Deriver + "\n"
	}

	if n.System != "" {
		out += "System: " + n.System + "\n"
	}

	for _, sig := range n.Signatures {
		out += "Sig: " + sig + "\n"
	}

	if n.CA != "" {
		out += "CA: " + n.CA + "\n"
	}

	return out
}

// ContentType returns the mime content type of the object
func (n narInfo) ContentType() string {
	return "text/x-nix-narinfo"
}
