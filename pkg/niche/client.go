package niche

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/colemickens/niche/pkg/narenc"
	"github.com/colemickens/niche/pkg/nixclient"
	"github.com/graymeta/stow"
	"github.com/rs/zerolog/log"
	"go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"
	"go.mozilla.org/sops/v3/version"
)

const wkPrivateConfig string = "niche.private.json"
const wkPublicConfig string = "niche.json"
const nicheCacheInfoPath string = "nix-cache-info"

type nicheClient struct {
	config        privateNicheConfig
	stowClient    stow.Location
	stowContainer stow.Container
	nix           nixclient.NixClient
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
	return clientFromBytes(byts)
}

// TODO: change this to a string so that we move URL validation
// here, and/or support file:////tmp/stow-local-for-golang-integration-tests
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
	encryptedStr := string(encryptedBytes)
	_ = encryptedStr
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
	return clientFromPrivateNicheConfig(cfg, false)
}

func clientFromPrivateNicheConfig(cfg privateNicheConfig, create bool) (*nicheClient, error) {
	if cfg.StorageKind == "fake" {
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, err
		}
		cfg.StorageKind = "local"
		cfg.StorageConfigMap = map[string]string{
			"path": tempDir,
		}
	}

	loc, err := stow.Dial(cfg.StorageKind, stow.ConfigMap(cfg.StorageConfigMap))
	if err != nil {
		return nil, err
	}

	var cntr stow.Container
	if create {
		if _, err := loc.Container(cfg.StorageContainer); err == nil {
			return nil, fmt.Errorf("container '%s' already exists", cfg.StorageContainer)
		}
		log.Info().Str("container", cfg.StorageContainer).Msgf("Creating storage container")
		cntr, err = loc.CreatePublicContainer(cfg.StorageContainer, false)
		if err != nil {
			return nil, err
		}
	} else {
		log.Trace().Str("container", cfg.StorageContainer).Msgf("Looking up storage container")
		cntr, err = loc.Container(cfg.StorageContainer)
		if err != nil {
			return nil, err
		}
	}

	newClient := &nicheClient{
		nix:           nixclient.NixClientCli{},
		config:        cfg,
		stowClient:    loc,
		stowContainer: cntr,
	}

	return newClient, nil
}

// TODO: this needs to re-generate public config and push it...
func (c *nicheClient) reuploadConfig() error {
	privCfgBytesRaw, err := json.Marshal(c.config)
	if err != nil {
		return err
	}
	log.Info().Msg("encrypting config with sops")
	encrPrivCfg, err := c.sopsEncrypt(privCfgBytesRaw)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(encrPrivCfg)
	log.Info().Msg("uploading private niche config")
	_, err = c.stowContainer.Put(wkPrivateConfig, buf, int64(len(encrPrivCfg)), nil)
	if err != nil {
		return err
	}

	pc := publicNicheConfig{
		PublicKey: c.config.PublicKey,
	}
	publicBytes, err := json.Marshal(pc)
	if err != nil {
		return err
	}
	log.Info().Msg("uploading public niche config")
	_, err = c.stowContainer.Put(wkPublicConfig, bytes.NewBuffer(publicBytes), int64(len(publicBytes)), nil)
	if err != nil {
		return err
	}

	cacheInfoBytes := []byte("StoreDir: /nix/store\nWantMassQuery: 1\nPriority: 40")
	log.Info().Msg("uploading nix-cache-info")
	_, err = c.stowContainer.Put(nicheCacheInfoPath, bytes.NewBuffer(cacheInfoBytes), int64(len(cacheInfoBytes)), nil)
	if err != nil {
		return err
	}

	return nil
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

func (c *nicheClient) ensurePath(storePath string) error {
	niItemPath, err := narinfoItemPath(storePath)
	if err != nil {
		return err
	}

	// TODO: extract this to "checkPath" function
	for _, server := range c.config.UpstreamServers {
		serverNarInfoURL, err := preprocessHostArg(server)
		if err != nil {
			return err // what do we do here, just fall-through?
		}
		s := serverNarInfoURL.String()

		serverNarInfoURL.Path = path.Join(serverNarInfoURL.Path, niItemPath)
		resp, err := http.Head(serverNarInfoURL.String())
		if err != nil {
			return err // what do we do here, just fall-through?
		}
		if resp.StatusCode == 200 {
			log.Trace().Str("storePath", storePath).Str("server", s).Msg("path skipped")
			return nil
		}
	}

	// we don't have the user-specified end-url, and don't really care to here
	// so form a nice string to indicate we're checking the actual niche cache now
	nicheServer := c.config.StorageKind + "/" + c.config.StorageContainer

	log.Trace().Str("storePath", storePath).Str("server", nicheServer).Msg("checking for path")
	_, errNarInfo := c.stowContainer.Item(niItemPath)
	if errNarInfo == nil {
		log.Trace().Str("storePath", storePath).Str("server", nicheServer).Msg("path skipped")
		return nil
	}

	log.Trace().Str("storePath", storePath).Msg("narinfo missing")
	return c.uploadPath(storePath)
}

func (c *nicheClient) uploadPath(storePath string) error {
	// if the narinfo exists, assume that we're in a good state
	// since we always upload the narinfo last
	niItemPath, err := narinfoItemPath(storePath)
	if err != nil {
		return err
	}

	//compressedNarFilePath, err := c.nix.DumpPath(storePath)
	compressedNarFilePath, err := narenc.DumpPathXz(storePath)
	if err != nil {
		return err
	}
	defer func() {
		os.Remove(compressedNarFilePath)
	}()

	narFile, err := os.Open(compressedNarFilePath)
	if err != nil {
		return err
	}

	pathInfo, err := c.nix.PathInfo(storePath)
	if err != nil {
		return err
	}

	narInfo, err := narInfoForNarFile(*pathInfo, compressedNarFilePath)

	err = narInfo.AddSignature(c.config.SigningKey)
	if err != nil {
		return err
	}

	// Upload
	_, err = c.stowContainer.Put(narInfo.URL, narFile, narInfo.NarSize, nil)
	if err != nil {
		return err
	}
	narInfoStr := narInfo.String()
	narInfoRdr := bytes.NewBufferString(narInfoStr)
	_, err = c.stowContainer.Put(niItemPath, narInfoRdr, int64(len(narInfoStr)), nil)
	if err != nil {
		return err
	}
	log.Info().Str("storePath", storePath).Msg("uploaded path")

	return nil
}
