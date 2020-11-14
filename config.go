package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
)

type nicheKeyGroup map[string][]string

type privateNicheConfig struct {
	SigningKey       string            `json:"signingKey"`
	PublicKey        string            `json:"publicKey"`
	StorageKind      string            `json:"storageKind"`      // TODO: stow's Kind
	StorageContainer string            `json:"storageContainer"` // TODO: stow's Kind
	StorageConfigMap map[string]string `json:"storageConfigMap"` // lazy, just let it hand off to Stow
	KeyGroups        []nicheKeyGroup   `json:"keyGroups"`
}

const defaultEditor = "nvim"

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

func fromEnvOrDefault(varname string) string {
	result := os.Getenv(varname)
	if result == "" {
		return "__" + varname + "__"
	}
	return result
}

func getInitialStorageConfigMap(kind string) (map[string]string, error) {
	switch kind {
	case "azure":
		return map[string]string{
			"account": fromEnvOrDefault("AZURE_STORAGE_ACCOUNT_NAME"),
			"key":     fromEnvOrDefault("AZURE_STORAGE_ACCESS_KEY"),
		}, nil
	case "s3":
		return map[string]string{
			"auth_type":     fromEnvOrDefault("S3_AUTH_TYPE"),
			"access_key_id": fromEnvOrDefault("S3_ACCESS_KEY_ID"),
			"secret_key":    fromEnvOrDefault("S3_SECRET_KEY"),
			"region":        fromEnvOrDefault("S3_REGION"),
			"endpoint":      fromEnvOrDefault("S3_ENDPOINT"),
			"disable_ssl":   fromEnvOrDefault("S3_DISABLE_SSL"),
			"v2_signing":    fromEnvOrDefault("S3_SIGNING"),
		}, nil
	case "swift":
		return map[string]string{
			"username":        fromEnvOrDefault("SWIFT_USERNAME"),
			"key":             fromEnvOrDefault("SWIFT_KEY"),
			"tenant_name":     fromEnvOrDefault("SWIFT_TENANT_NAME"),
			"tenant_auth_url": fromEnvOrDefault("SWIFT_TENANT_AUTH_URL"),
		}, nil
	case "oracle":
		return map[string]string{
			"username":        fromEnvOrDefault("ORACLE_USERNAME"),
			"key":             fromEnvOrDefault("ORACLE_KEY"),
			"tenant_name":     fromEnvOrDefault("ORACLE_TENANT_NAME"),
			"tenant_auth_url": fromEnvOrDefault("ORACLE_TENANT_AUTH_URL"),
		}, nil
	case "google":
		return map[string]string{
			"username":        fromEnvOrDefault("GOOGLE_USERNAME"),
			"key":             fromEnvOrDefault("GOOGLE_KEY"),
			"tenant_name":     fromEnvOrDefault("GOOGLE_TENANT_NAME"),
			"tenant_auth_url": fromEnvOrDefault("GOOGLE_TENANT_AUTH_URL"),
		}, nil
	case "b2":
		return map[string]string{
			"username":        fromEnvOrDefault("B2_USERNAME"),
			"key":             fromEnvOrDefault("B2_KEY"),
			"tenant_name":     fromEnvOrDefault("B2_TENANT_NAME"),
			"tenant_auth_url": fromEnvOrDefault("B2_TENANT_AUTH_URL"),
		}, nil
	}

	return nil, fmt.Errorf("unsupported kind for init: %s", kind)
}
