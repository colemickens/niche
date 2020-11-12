package main

import (
	"io/ioutil"
	"os"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/config"
	"gopkg.in/yaml.v2"
)

type CreationRule struct {
	PathRegex string          `json:"path_regex" yaml:"path_regex"`
	KeyGroups []nicheKeyGroup `json:"key_groups" yaml:"key_groups"`
}

type Config struct {
	CreationRules []CreationRule `json:"creation_rules" yaml:"creation_rules"`
}

func keyGroupsFromKeyGroups(keyGroupsBlob []nicheKeyGroup) ([]sops.KeyGroup, error) {
	// construct fake file with creation rule to hold keygroups
	// this sucks, just to work around SOPS horrible api/codebase
	fakeConfig := Config{
		CreationRules: []CreationRule{
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
