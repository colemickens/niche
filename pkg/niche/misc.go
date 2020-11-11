package niche

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/colemickens/niche/pkg/narinfo"
	"github.com/colemickens/niche/pkg/nixb32"
	"github.com/colemickens/niche/pkg/nixclient"
	"github.com/rs/zerolog/log"
)

//const nixCharset string = "0123456789abcdfghijklmnpqrsvwxyz"

var slugRe *regexp.Regexp

func init() {
	slugRe = regexp.MustCompile("/nix/store/([0123456789abcdfghijklmnpqrsvwxyz]{32})(.*)")
}

//Wvar nix nixclient.NixClient = nixclient.NixClientCli{}

func preprocessHostArg(host string) (*url.URL, error) {
	if host == "" {
		return nil, fmt.Errorf("niche-url (-u) must be specified")
	}
	if !strings.HasPrefix(host, "https://") && !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	return url.Parse(host)
}

func narinfoItemPath(storePath string) (string, error) {
	matches := slugRe.FindStringSubmatch(storePath)
	// TOOD: better error handling
	narInfoItemPath := fmt.Sprintf("%s.narinfo", matches[1])
	return narInfoItemPath, nil
}

func hashFileToNixBase32(p string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(p)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatal().Err(err)
	}
	hashStr, err := nixb32.Hash(hasher.Sum(nil))
	if err != nil {
		log.Fatal().Err(err)
	}
	return hashStr, nil
}

func narInfoForNarFile(pathInfo nixclient.NixPathInfo, narFilePath string) (*narinfo.NarInfo, error) {
	stat, err := os.Stat(narFilePath)
	if err != nil {
		return nil, err
	}
	fileSize := stat.Size()

	fileHash, err := hashFileToNixBase32(narFilePath)
	if err != nil {
		return nil, err
	}

	prefixedHash := "sha256:" + fileHash

	references := make([]string, len(pathInfo.References))
	for i, ref := range pathInfo.References {
		references[i] = strings.TrimPrefix(ref, "/nix/store/") // TODO: prefix is hardcoded :(
		// TODO: note the trailing /
	}

	cmd := exec.Command("nix", "to-base32", pathInfo.NarHash)
	outBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	fixedNarHash := "sha256:" + strings.TrimSpace(string(outBytes))

	narURLPath := fmt.Sprintf("nars/%s.nar.xz", fileHash)
	narInfo := &narinfo.NarInfo{
		StorePath:   pathInfo.Path,
		URL:         narURLPath,
		NarHash:     fixedNarHash,
		NarSize:     pathInfo.NarSize,
		Compression: "xz",
		FileHash:    prefixedHash,
		FileSize:    fileSize,
		Deriver:     pathInfo.Deriver,
		//References:  pathInfo.References,
		References: references,
		CA:         "",
		//System:      "", //TODO
		Signatures: pathInfo.Signatures,
	}

	return narInfo, nil
}
