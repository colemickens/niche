package narinfo

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

type NarInfo struct {
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

func (ni NarInfo) String() string {
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
func (ni NarInfo) ContentType() string {
	return "text/x-nix-narinfo"
}

func (ni *NarInfo) AddSignature(privateKeyStr string) error {
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

func ensurePrefixed(storePath string) string {
	if !strings.HasPrefix(storePath, "/nix/store/") {
		storePath = "/nix/store/" + storePath
	}
	return storePath
}

func ensureBase32Hash(hash string) string {
	return hash
}

func (ni *NarInfo) Fingerprint() ([]byte, error) {
	storePath := ensurePrefixed(ni.StorePath)
	narHash := ensureBase32Hash(ni.NarHash)
	narSize := fmt.Sprintf("%d", ni.NarSize)

	prefixedRefs := make([]string, len(ni.References))
	for i, p := range ni.References {
		prefixedRefs[i] = ensurePrefixed(p)
	}
	references := strings.Join(prefixedRefs, ",")

	fp := strings.Join([]string{"1", storePath, narHash, narSize, references}, ";")
	return []byte(fp), nil
}
