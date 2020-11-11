package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"

	"go.mozilla.org/sops/v3/decrypt"
)

type versionedNicheConfig struct {
	ConfigVersion int    `json:"configVersion"`
	Config        []byte `json:"config"`
}

// TODO: do we want a public config section for "upstreams" (like cache.nixos.org)
// but most of what we need is private
type nicheConfigV1 struct {
	CacheURL         string            `json:"cacheUrl"`
	SigningKey       string            `json:"signingKey"`
	StorageKind      string            `json:"storageKind"`      // TODO: stow's Kind
	StorageContainer string            `json:"storageContainer"` // TODO: stow's Kind
	StorageConfigMap map[string]string `json:"storageConfigMap"` // lazy, just let it hand off to Stow
	SopsRecipients   []string          `json:"sopsRecipients"`
}

type unversionedNicheConfig nicheConfigV1

// DefaultEditor is vim because we're adults ;)
const defaultEditor = "vim"

// OpenFileInEditor opens filename in a text editor.
func OpenFileInEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = defaultEditor
	}

	// Get the full executable path for the editor.
	executable, err := exec.LookPath(editor)
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CaptureInputFromEditor opens a temporary file in a text editor and returns
// the written bytes on success or an error on failure. It handles deletion
// of the temporary file behind the scenes.
func CaptureInputFromEditor(initialContents []byte) ([]byte, error) {
	file, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return []byte{}, err
	}

	filename := file.Name()

	err = ioutil.WriteFile(filename, initialContents, 0644)
	if err != nil {
		return []byte{}, err
	}

	// Defer removal of the temporary file in case any of the next steps fail.
	defer os.Remove(filename)

	if err = file.Close(); err != nil {
		return []byte{}, err
	}

	if err = OpenFileInEditor(filename); err != nil {
		return []byte{}, err
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return []byte{}, err
	}

	return bytes, nil
}

func editConfig(baseURL url.URL) (*nicheConfigV1, error) {
	// fetch
	configURL := baseURL
	configURL.Path = path.Join(baseURL.Path, "/.well-known/niche-config")

	resp, err := http.Get(configURL.String())
	if err != nil {
		return nil, err
	}

	encryptedBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	decryptedVersionedConfig, err := decrypt.Data(encryptedBodyBytes, "binary")
	if err != nil {
		return nil, err
	}

	// write bytes to temp file
	// open editor with bytes
	newConfigData, err := CaptureInputFromEditor(decryptedVersionedConfig)
	if err != nil {
		return nil, err
	}

	_ = newConfigData

	// re-encrypt with sops
	// data, err := encrypt.Data(newConfigData)
	// if err != nil {
	// 	return nil, err
	// }

	//return data, nil
	return nil, nil // TODO:::::::::::::::::::::::::::::::::::;
}

func configFromFilePath(path string) (unversionedNicheConfig, error) {
	return unversionedNicheConfig{}, nil
}

func replaceConfig(cache url.URL, config unversionedNicheConfig) error {
	return nil
}

// func replaceConfig() {

// 	argConfig.cacheURL = args[0]

// 	testBlob := []byte(`{ "pgp": [ "abc", "def" ] }`)

// 	var data map[string]interface{}
// 	err := json.Unmarshal(testBlob, &data)
// 	if err != nil {
// 		return err
// 	}

// 	keyGroups, err := keyGroupsFromKeyGroups(data)
// 	if err != nil {
// 		panic(err)
// 		return err
// 	}

// 	_ = keyGroups

// 	configFileBytes, err := ioutil.ReadFile(argConfig.configFilePath)
// 	if err != nil {
// 		return err
// 	}

// 	encryptedConfigFile, err := encrypt(configFileBytes, keyGroups)
// 	if err != nil {
// 		return err
// 	}

// 	var cfg nicheConfigV1
// 	// TODO: read from the config file to populate this so we can use the values too

// 	// TODO: convert map[string]interface{} to map[string]string
// 	location, err := stow.Dial(cfg.StorageKind, stow.ConfigMap(cfg.StorageConfigMap))
// 	if err != nil {
// 		return err
// 	}
// 	defer location.Close()

// 	container, err := location.Container(cfg.StorageContainer)
// 	if err != nil {
// 		return err
// 	}

// 	buf := bytes.NewReader(encryptedConfigFile)
// 	var size int64 = int64(len(encryptedConfigFile))
// 	_, err = container.Put("/.well-known/niche-config", buf, size, nil)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func somethingThatUsedParsedConfig() error {
// 	var config versionedNicheConfig
// 	err := json.Unmarshal(decryptedVersionedConfig, &config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if config.ConfigVersion != 1 {
// 		return fmt.Errorf("unsupported version")
// 	}

// 	var configV1 nicheConfigV1
// 	err = json.Unmarshal(config.Config, &configV1)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &configV1, nil
// }
