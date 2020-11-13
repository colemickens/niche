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
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/graymeta/stow"
	_ "github.com/graymeta/stow/azure"
	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"
	"go.mozilla.org/sops/v3/version"
)

const wkPrivateConfig = ".well-known/niche-private-config"

type nicheClient struct {
	config        privateNicheConfig
	stowClient    stow.Location
	stowContainer stow.Container
}

func clientFromFile(pth string) (*nicheClient, error) {
	f, err := os.Open(pth)
	if err != nil {
		return nil, err
	}
	byts, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(byts))
	return clientFromBytes(byts)
}

func clientFromSops(cacheURL url.URL) (*nicheClient, error) {
	// copy cacheURL and join path wi/ the config path-suffix
	privateNicheConfigURL := cacheURL
	privateNicheConfigURL.Path = path.Join(privateNicheConfigURL.Path, wkPrivateConfig)
	resp, err := http.Get(privateNicheConfigURL.String())
	if err != nil {
		return nil, err
	}
	encryptedBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	decryptedBytes, err := decrypt.Data(encryptedBytes, "binary")
	if err != nil {
		return nil, err
	}
	return clientFromBytes(decryptedBytes)
}

func clientFromBytes(byts []byte) (*nicheClient, error) {
	var cfg privateNicheConfig
	err := json.Unmarshal(byts, &cfg)
	if err != nil {
		return nil, err
	}
	return clientFromPrivateNicheConfig(cfg)
}

func clientFromPrivateNicheConfig(cfg privateNicheConfig) (*nicheClient, error) {
	loc, err := stow.Dial(cfg.StorageKind, stow.ConfigMap(cfg.StorageConfigMap))
	if err != nil {
		return nil, err
	}

	cntr, err := loc.Container(cfg.StorageContainer)
	if err != nil {
		return nil, err
	}

	newClient := &nicheClient{
		config:        cfg,
		stowClient:    loc,
		stowContainer: cntr,
	}

	return newClient, nil
}

func (c *nicheClient) reuploadConfig() error {
	decryptedNewCfgBytes, err := json.Marshal(c.config)
	if err != nil {
		return err
	}
	encNewCfgBytes, err := c.sopsEncrypt(decryptedNewCfgBytes)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(encNewCfgBytes)
	_, err = c.stowContainer.Put(wkPrivateConfig, buf, int64(len(encNewCfgBytes)), nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *nicheClient) signNarPrint(print string) (string, error) {
	return "", nil
}

func (c *nicheClient) sopsEncrypt(fileBytes []byte) ([]byte, error) {
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
	dataKey, errs := tree.GenerateDataKey()
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

func (c *nicheClient) ensurePath(storePath string, alwaysOverwrite bool) error {
	narPath, infoPath := narPathsFromStorePath(storePath)

	_, errNarXz := c.stowContainer.Item(narPath)
	_, errNarInfo := c.stowContainer.Item(infoPath)

	if errNarXz != nil || errNarInfo != nil || alwaysOverwrite {
		// we need the NAR hash if we don't have (it in) the narinfo
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
		narSize := stat.Size()
		narFile, err := os.Open(compressedNarFilePath)
		if err != nil {
			return err
		}
		narItem, err := c.stowContainer.Put(narPath, narFile, narSize, nil)
		if err != nil {
			return err
		}
		fmt.Println("uploaded .nar.xz:", narItem)

		hashCmd := exec.Command("nix", "hash-file", storePath)
		hashBytes, err := hashCmd.Output()
		fileHash := strings.TrimSpace(string(hashBytes))

		narInfo, err := narInfoForPath(storePath, narPath, fileHash, narSize)
		if err != nil {
			return err
		}
		err = narInfo.AddSignature(c.config.SigningKey)
		if err != nil {
			return err
		}

		narInfoStr := narInfo.String()
		narInfoRdr := bytes.NewBufferString(narInfoStr)
		infoItem, err := c.stowContainer.Put(infoPath, narInfoRdr, int64(len(narInfoStr)), nil)
		if err != nil {
			return err
		}
		fmt.Println("uploaded narinfo:", infoItem)
	}
	return nil
}

func ensureNarInfoContainsSig(ni *narInfo, sig string) {
	// TODO: better logic here
	ni.Signatures = []string{sig}
}

func narPathsFromStorePath(storePath string) (string, string) {
	storePathBase := filepath.Base(storePath)
	narPath := fmt.Sprintf("nars/%s.nar.xz", storePathBase)
	infoPath := storePathBase + ".narinfo"
	return narPath, infoPath
}
