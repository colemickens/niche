package main

import (
	"os"
	"strings"

	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
	"github.com/rs/zerolog/log"
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

func configFieldsForKind(kind string) []string {
	switch kind {
	case "azure":
		return []string{"account", "key"}
	case "s3":
		return []string{"auth_type", "access_key_id", "secret_key", "region", "endpoint", "disable_ssl", "v2_signing"}
	case "swift":
		return []string{"username", "key", "tenant_name", "tenant_auth_url"}
	case "oracle":
		return []string{"username", "password", "authorization_endpoint"}
	case "google":
		return []string{"json", "project_id", "scope"}
	case "b2":
		return []string{"account_id", "application_key", "application_key_id"}
	}
	log.Fatal().Msg("invalid storage kind")
	return nil
}

func getInitialStorageConfigMap(kind string) map[string]string {
	fields := configFieldsForKind(kind)
	result := make(map[string]string)
	missingVals := []string{}
	for _, fieldName := range fields {
		envVarName := strings.ToUpper(kind + "_" + fieldName)
		val, found := os.LookupEnv(envVarName)
		if !found {
			missingVals = append(missingVals, envVarName)
		}
		result[fieldName] = val
	}
	if len(missingVals) > 0 {
		log.Fatal().Str("kind", kind).
			Strs("missingEnvVars", missingVals).
			Msgf("missing required env vars to init config")
	}
	return result
}
