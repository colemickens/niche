package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.mozilla.org/sops/v3/decrypt"

	"github.com/graymeta/stow"
	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
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
	SopsRecipients   []string          `json: "sopsRecipients"`
}

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

func signPath() {

}

func uploadPath() {

}

func echoServer(c net.Conn) {
	log.Printf("Client connected [%s]", c.RemoteAddr().Network())
	io.Copy(c, c)
	// TODO how to read lines?
	// loop
	// on "HANGUP", we disconnect, delete socket
	c.Close()
}

func listen(socketPath string, cacheURL url.URL) error {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	for {
		// Accept new connections, dispatching them to echoServer
		// in a goroutine.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}

		go echoServer(conn)
	}
	defer os.RemoveAll(socketPath)
	return nil
}

func main() {
	var rootCmd = &cobra.Command{Use: "app"}

	var argListen struct {
		socketPath string
		cache      string

		// parsed out:
		cacheURL *url.URL
	}
	var cmdListen = &cobra.Command{
		Use:  "listen [{--socket | -s} socket_path] [cache]",
		Args: cobra.MinimumNArgs(1), // TODO: do the URL parsing here instead
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			argListen.cacheURL, err = url.Parse(args[0])
			if err != nil {
				return err
			}

			if argListen.socketPath == "" {
				dir, err := ioutil.TempDir("", "niche")
				if err != nil {
					return err
				}
				defer os.RemoveAll(dir)
				argListen.socketPath = filepath.Join(dir, "queue.sock")
			}

			err = listen(argListen.socketPath, *argListen.cacheURL)
			if err != nil {
				return err
			}

			fmt.Println(argListen.socketPath)
			return nil
		},
	}
	cmdListen.PersistentFlags().StringVarP(&argListen.socketPath, "socket", "s", "", "path for socket to listen on")
	rootCmd.AddCommand(cmdListen)

	var cmdQueue = &cobra.Command{
		Use:  "queue [socket_path] [path]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			socketPath := args[0]
			paths := args[1 : len(args)-1]

			_ = socketPath
			_ = paths
			// TODO: support --stdin
			// TODO: support --from-result
			// TODO: support --from-file
			// err := enqueuePath(socketPath, path)
			// if err != nil {
			// 	return err
			// }
			return nil
		},
	}
	rootCmd.AddCommand(cmdQueue)

	var argConfig struct {
		cacheURL       string
		configFilePath string
	}
	var cmdConfig = &cobra.Command{
		Use:  "config [{--config-file | -c} config_file] [cache_url]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			argConfig.cacheURL = args[0]

			testBlob := []byte(`{ "pgp": [ "abc", "def" ] }`)

			var data map[string]interface{}
			err := json.Unmarshal(testBlob, &data)
			if err != nil {
				return err
			}

			keyGroups, err := keyGroupsFromKeyGroups(data)
			if err != nil {
                    panic(err)
				return err
			}

			_ = keyGroups

			configFileBytes, err := ioutil.ReadFile(argConfig.configFilePath)
			if err != nil {
				return err
			}

			encryptedConfigFile, err := encrypt(configFileBytes, keyGroups)
			if err != nil {
				return err
			}

			var cfg nicheConfigV1
			// TODO: read from the config file to populate this so we can use the values too

			// TODO: convert map[string]interface{} to map[string]string
			location, err := stow.Dial(cfg.StorageKind, stow.ConfigMap(cfg.StorageConfigMap))
			if err != nil {
				return err
			}
			defer location.Close()

			container, err := location.Container(cfg.StorageContainer)
			if err != nil {
				return err
			}

			buf := bytes.NewReader(encryptedConfigFile)
			var size int64 = int64(len(encryptedConfigFile))
			_, err = container.Put("/.well-known/niche-config", buf, size, nil)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmdConfig.PersistentFlags().StringVarP(&argConfig.configFilePath, "config-file", "c", "", "path to config file to init/force overwrite")
	rootCmd.AddCommand(cmdConfig)

	rootCmd.Execute()
	// niche --cache https://cache.nixcache.org

	// donwload /.well-known/niche
	// decrypt with sops

	// use the 'provider' key to switch
	// get a slop client

	// check if the path exists on remote, if not, sign+push
	// TODO: how to combine sigs if we have a NAR with multiple sigs
	// TODO: do we need to protect against non-/nix/store prefix?

	// niche upload https://cache.niche.org

	// niche listen /tmp/niche.socket https://cache.niche.org
	// echo "/nix/store/test" | niche --queue /tmp/niche.socket

	// niche update-config ./niche-config https://cache.niche.org

}
