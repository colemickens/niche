package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

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

// TODO: I probably need lots of locking around the client here since it's getting hit up from multiple places

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
	// TODO: which first, maybe better to upload nar.xz then narinfo so we're sure
	// we can treat narinfo as sentinel?
	//
	// TODO:
	// - it might be more approrpriate to:
	//  - get narinfo from server
	//  - follow URL field to NAR
	//  - check NAR exists
	// - if not,
	//  - hash NAR
	//  - use NAR hash to write URL/FileHash into the narinfo
	//  - this way we follow how cachix.org narinfo files look
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
		fmt.Println("uploaded .nar.xz:", narItem.Name())

		fileHash, err := nixHashFile(storePath)

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
		fmt.Println("uploaded narinfo:", infoItem.Name())
	}
	return nil
}

// TODO: we need to write to a single queue
// right now each build client get its own queue
// which is also what cachix does and it seems bad
func (c *nicheClient) listen(socketPath string, queue chan string) error {
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go handle(c, conn, queue)
	}
}

func handle(c *nicheClient, conn net.Conn, queue chan string) {
	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
	}()

	//timeoutDuration := 1000 * time.Second // TODO?
	bufReader := bufio.NewReader(conn)

	for {
		//conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		byts, err := bufReader.ReadBytes('\n')
		if err != nil {
			log.Println("uhhh BAD", err)
			break
		}
		storePath := strings.TrimSpace(string(byts))
		log.Println("received", storePath)

		if storePath == "QUIT" {
			log.Println("told to quit")
			queue <- storePath
			break
		}

		allStorePaths, err := getAllStorePaths(storePath)
		if err != nil {
			log.Println("uhhh BAD", err)
			break
		}

		for _, storePath := range allStorePaths {
			log.Println("propagating", storePath)
			queue <- storePath
		}
	}
}

func (c *nicheClient) processBuildQueue(queue chan string, wg *sync.WaitGroup, alwaysOverwrite bool) {
	wg.Add(1)
	defer wg.Done()

	seenPaths := []string{}

	for storePath := range queue {
		if storePath == "QUIT" {
			log.Println("leaving build queue")
			return
		}
		for _, seenPath := range seenPaths {
			if strings.EqualFold(storePath, seenPath) {
				continue
			}
		}
		c.ensurePath(storePath, alwaysOverwrite)
		seenPaths = append(seenPaths, storePath)
		log.Println("ensured", storePath)
	}
}

func narPathsFromStorePath(storePath string) (string, string) {
	storePathBase := filepath.Base(storePath)
	narPath := fmt.Sprintf("nars/%s.nar.xz", storePathBase)
	infoPath := storePathBase + ".narinfo"
	// TODO: write test for this (that should currently fail)
	return narPath, infoPath
}
