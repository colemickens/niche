package niche

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/config"
	"gopkg.in/yaml.v2"
)

type sopsCreationRule struct {
	PathRegex string          `json:"path_regex" yaml:"path_regex"`
	KeyGroups []nicheKeyGroup `json:"key_groups" yaml:"key_groups"`
}

type sopsConfig struct {
	CreationRules []sopsCreationRule `json:"creation_rules" yaml:"creation_rules"`
}

func keyGroupsFromKeyGroups(keyGroupsBlob []nicheKeyGroup) ([]sops.KeyGroup, error) {
	// construct fake file with creation rule to hold keygroups (just to get around SOPS)
	fakeConfig := sopsConfig{
		CreationRules: []sopsCreationRule{
			{
				PathRegex: ".*$",
				KeyGroups: keyGroupsBlob,
			},
		},
	}
	tmpfile, err := ioutil.TempFile("", "temp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpfile.Name()) // clean up

	fakeConfigBytes, err := yaml.Marshal(fakeConfig)
	if err != nil {
		return nil, err
	}

	if _, err := tmpfile.Write(fakeConfigBytes); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}

	config, err := config.LoadCreationRuleForFile(tmpfile.Name(), "somefilename.txt", nil)
	if err != nil {
		return nil, err
	}

	return config.KeyGroups, nil
}

func generateBinaryCacheKeys(cacheName string) (string, string, error) {
	pubKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}

	privateKeyStr := base64.StdEncoding.EncodeToString(privateKey)
	publicKeyStr := base64.StdEncoding.EncodeToString(pubKey)

	finalPrivateKeyStr := cacheName + ":" + privateKeyStr
	finalPublicKeyStr := cacheName + ":" + publicKeyStr

	return finalPrivateKeyStr, finalPublicKeyStr, nil
}
