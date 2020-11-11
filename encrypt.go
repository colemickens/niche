package main

import (
	"fmt"
	"io/ioutil"
	"os"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/config"
	"go.mozilla.org/sops/v3/keyservice"
	"go.mozilla.org/sops/v3/version"
	"gopkg.in/yaml.v2"
)

type KG map[string]interface{}

type CreationRule struct {
	PathRegex string `json:"path_regex" yaml:"path_regex"`
	KeyGroups []KG   `json:"key_groups" yaml:"key_groups"`
}

type Config struct {
	CreationRules []CreationRule `json:"creation_rules" yaml:"creation_rules"`
}

func keyGroupsFromKeyGroups(keyGroupsBlob KG) ([]sops.KeyGroup, error) {
	// construct fake file with creation rule to hold keygroups
	// this sucks, just to work around SOPS horrible api/codebase
	fakeConfig := Config{
		CreationRules: []CreationRule{
			{
				PathRegex: ".*$",
				KeyGroups: []KG{keyGroupsBlob},
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

func encrypt(fileBytes []byte, keyGroups []sops.KeyGroup) (encryptedFileBytes []byte, err error) {
	inputStore := common.StoreForFormat(formats.FormatFromString("binary"))
	outputStore := common.StoreForFormat(formats.FormatFromString("binary"))
	branches, err := inputStore.LoadPlainFile(fileBytes)
	if err != nil {
		return nil, err
	}
	// path, err := filepath.Abs(opts.InputPath)
	// if err != nil {
	// 	return nil, err
	// }
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:         keyGroups,
			UnencryptedSuffix: "",
			EncryptedSuffix:   "",
			UnencryptedRegex:  "",
			EncryptedRegex:    "",
			Version:           version.Version,
			ShamirThreshold:   0,
		},
		FilePath: "non-applicable",
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{})
	if len(errs) > 0 {
		err = fmt.Errorf("Could not generate data key: %s", errs)
		return nil, err
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  aes.NewCipher(),
	})
	if err != nil {
		return nil, err
	}

	encryptedFileBytes, err = outputStore.EmitEncryptedFile(tree)
	if err != nil {
		return nil, err
	}
	return encryptedFileBytes, nil
}
