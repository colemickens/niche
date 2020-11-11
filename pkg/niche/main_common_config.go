package niche

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

const defaultUpstreamServers string = "https://cache.nixos.org" // idk, use a string since go doesn't have const lists, comma-separated

func configInit(cacheName, kind, container string, fingerprints []string) error {
	if cacheName == "" || kind == "" || container == "" || len(fingerprints) == 0 {
		log.Fatal().
			Str("cacheName", cacheName).
			Str("kind", kind).
			Str("container", container).
			Strs("fingerprints", fingerprints).
			Msg("cache-name, kind, fingerprint, and container must all be specified")
	}
	log.Info().
		Str("cacheName", cacheName).
		Str("kind", kind).
		Str("container", container).
		Strs("fingerprints", fingerprints).
		Msg("config_init")
	var privateKeyStr string
	var publicKeyStr string
	var found bool
	var err error

	privateKeyStr, found = os.LookupEnv("NICHE_SIGNING_KEY")
	if found {
		pubFromPrivateKeyStr := func(s string) (string, error) {
			return "", nil
		}
		publicKeyStr, err = pubFromPrivateKeyStr(privateKeyStr)
		if err != nil {
			return err
		}
		log.Info().Str("publicKey", publicKeyStr).Msgf("Using signingkey from ${NICHE_SIGNING_KEY}")
	} else {
		log.Info().Msgf("Generating new signing key")
		if privateKeyStr, publicKeyStr, err = generateBinaryCacheKeys(cacheName); err != nil {
			return err
		}
		log.Info().Str("publicKey", publicKeyStr).Msgf("Generated new signing key")
	}

	// create stow's config map from expected env vars
	configMap := getInitialStorageConfigMap(kind)

	newConfig := privateNicheConfig{
		StorageKind:      kind,
		SigningKey:       privateKeyStr,
		PublicKey:        publicKeyStr,
		StorageContainer: container,
		StorageConfigMap: configMap,
		KeyGroups:        []nicheKeyGroup{{"pgp": fingerprints}},
		UpstreamServers:  strings.Split(defaultUpstreamServers, ","),
	}

	c, err := clientFromPrivateNicheConfig(newConfig, true)
	if err != nil {
		return err
	}

	err = c.reuploadConfig()
	if err != nil {
		return err
	}

	log.Info().Msg("successfully created new niche repo")
	// TODO: get Stow to have a NiceURL() that we can use here?

	return nil
}

func configDownload(cacheNameRaw, configFilePath string) error {
	cacheURL, err := preprocessHostArg(cacheNameRaw)
	if err != nil {
		return err
	}

	if configFilePath == "" {
		log.Fatal().Msg("'config' argument is required")
	}

	c, err := clientFromSops(*cacheURL)
	if err != nil {
		return err
	}
	defer c.stowClient.Close()

	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return err
	}
	ioutil.WriteFile(configFilePath, data, 0600)

	log.Info().Str("configFile", configFilePath).Msg("config download complete")

	return nil
}

func configUpload(configFilePath string) error {
	c, err := clientFromFile(configFilePath)
	if err != nil {
		return err
	}
	defer c.stowClient.Close()

	err = c.reuploadConfig()
	if err != nil {
		return err
	}

	log.Info().Str("configFile", configFilePath).Msg("config upload complete")

	return nil
}
