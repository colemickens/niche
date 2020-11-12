package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/graymeta/stow"
	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"
	"go.mozilla.org/sops/v3/keyservice"
	"go.mozilla.org/sops/v3/version"
)

const WK_PRIVATE_CONFIG = ".well-known/niche-private-config"

type Client struct {
	config        privateNicheConfig
	stowClient    stow.Location
	stowContainer stow.Container
}

func clientFromFile(pth string) (*Client, error) {
	f, err := os.Open(pth)
	if err != nil {
		return nil, err
	}
	byts, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return clientFromBytes(byts)
}

func clientFromSops(cacheURL url.URL) (*Client, error) {
	// copy cacheURL and join path wi/ the config path-suffix
	privateNicheConfigURL := cacheURL
	privateNicheConfigURL.Path = path.Join(privateNicheConfigURL.Path, WK_PRIVATE_CONFIG)
	// http get
	resp, err := http.Get(privateNicheConfigURL.String())
	if err != nil {
		return nil, err
	}
	// read the resp
	encryptedBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// decrypt with sops
	decryptedBytes, err := decrypt.Data(encryptedBytes, "binary")
	if err != nil {
		return nil, err
	}
	return clientFromBytes(decryptedBytes)
}

func clientFromBytes(byts []byte) (*Client, error) {
	var cfg privateNicheConfig
	err := json.Unmarshal(byts, &cfg)
	if err != nil {
		return nil, err
	}
	return clientFromPrivateNicheConfig(cfg)
}

func clientFromPrivateNicheConfig(cfg privateNicheConfig) (*Client, error) {
	loc, err := stow.Dial(cfg.StorageKind, stow.ConfigMap(cfg.StorageConfigMap))
	if err != nil {
		return nil, err
	}

	cntr, err := loc.Container(cfg.StorageContainer)
	if err != nil {
		return nil, err
	}

	newClient := &Client{
		config:        cfg,
		stowClient:    loc,
		stowContainer: cntr,
	}

	return newClient, nil
}

func (c *Client) reuploadConfig() error {
	decryptedNewCfgBytes, err := json.Marshal(c.config)
	if err != nil {
		return err
	}

	encNewCfgBytes, err := c.sopsEncrypt(decryptedNewCfgBytes)
	if err != nil {
		return err
	}
	// TOOD: re-encrypt!!!!
	buf := bytes.NewBuffer(encNewCfgBytes)
	_, err = c.stowContainer.Put(WK_PRIVATE_CONFIG, buf, int64(len(encNewCfgBytes)), nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) signNarPrint(print string) (string, error) {
	return "", nil
}

func (c *Client) sopsEncrypt(fileBytes []byte) ([]byte, error) {
	inputStore := common.StoreForFormat(formats.FormatFromString("binary"))
	outputStore := common.StoreForFormat(formats.FormatFromString("binary"))
	sopsKeyGroups, err := keyGroupsFromKeyGroups(c.config.KeyGroups)
	if err != nil {
		return nil, err
	}
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
			KeyGroups:         sopsKeyGroups,
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

	encryptedFileBytes, err := outputStore.EmitEncryptedFile(tree)
	if err != nil {
		return nil, err
	}
	return encryptedFileBytes, nil
}

func (c *Client) ensurePath(storePath string) error {
	narXzItemName := narXzKeyForPath(c.config.StorageContainer, storePath)
	narInfoItemName := narInfoKeyForPath(c.config.StorageContainer, storePath)

	_, errNarXz := c.stowContainer.Item(narXzItemName)
	_, errNarInfo := c.stowContainer.Item(narInfoItemName)

	if errNarXz != nil {
		compressedNarFilePath, err := nixDumpPath(storePath)
		if err != nil {
			return err
		}
		defer func() {
			log.Println("removing", compressedNarFilePath)
			os.Remove(compressedNarFilePath)
		}()
		stat, err := os.Stat(compressedNarFilePath)
		if err != nil {
			return err
		}
		f, err := os.Open(compressedNarFilePath)
		if err != nil {
			return err
		}
		item, err := c.stowContainer.Put(narXzItemName, f, stat.Size(), nil)
		if err != nil {
			return err
		}
		fmt.Println("uploaded", item)
	}

	if errNarInfo != nil {
		signature := "" // TODO

		narInfo, err := nixPathInfo(storePath)
		if err != nil {
			return err
		}

		narInfo.Signatures = []string{signature}
		narInfo.Compression = "xz"
		narInfo.URL = "TODO TODO TODO"
		// narInfo.{System,FileSize,FileHash} ??

		narInfoStr := narInfo.String()
		narInfoRdr := bytes.NewBufferString(narInfoStr)
		item, err := c.stowContainer.Put(narXzItemName, narInfoRdr, int64(len(narInfoStr)), nil)
		if err != nil {
			return err
		}
		fmt.Println("uploaded", item)
	}
	return nil
}

func narInfoKeyForPath(containerName, storePath string) string {
	return containerName + "/" + storePath + ".narinfo"
}

func narXzKeyForPath(containerName, storePath string) string {
	return containerName + "/nars/" + storePath + ".nar.xz"
}
