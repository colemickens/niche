package niche

import (
	"os"
	"strings"

	_ "github.com/graymeta/stow/google"
	_ "github.com/graymeta/stow/s3"
	"github.com/rs/zerolog/log"
)

type nicheKeyGroup map[string][]string

// TODO: where to store the values used in `nix-cache-info`?
type privateNicheConfig struct {
	TestMode         bool              `json:"testMode"`
	SigningKey       string            `json:"signingKey"`
	PublicKey        string            `json:"publicKey"`
	StorageKind      string            `json:"storageKind"`      // TODO: stow's Kind
	StorageContainer string            `json:"storageContainer"` // TODO: stow's Kind
	StorageConfigMap map[string]string `json:"storageConfigMap"` // lazy, just let it hand off to Stow
	KeyGroups        []nicheKeyGroup   `json:"keyGroups"`
	UpstreamServers  []string          `json:"upstreamServers"`
}

type publicNicheConfig struct {
	PublicKey string `json:"publicKey"`

	// this would be used to prevent re-uploading things already in cache.nixos.org/nixpkgs-wayland.cachix.org
	// and also maybe for `niche build -n` (no upload)
	//UpstreamServers string `json:"upstreamServers"`
	//UpstreamKeys    string `json:"upstreamKeys"`
}

func configFieldsForKind(kind string) ([]string, []string) {
	switch kind {
	case "azure":
		return []string{"account", "key"}, []string{}
	case "s3":
		return []string{"access_key_id", "secret_key"}, []string{"auth_type", "region", "endpoint", "disable_ssl", "v2_signing"}
	case "swift":
		return []string{"username", "key", "tenant_name", "tenant_auth_url"}, []string{}
	case "oracle":
		return []string{"username", "password", "authorization_endpoint"}, []string{}
	case "google":
		return []string{"json", "project_id", "scope"}, []string{}
	case "b2":
		return []string{"account_id", "application_key", "application_key_id"}, []string{}
	case "local":
		return []string{"path"}, []string{}
	}

	log.Fatal().Str("kind", kind).Msg("invalid storage kind")
	return nil, nil
}

func getInitialStorageConfigMap(kind string) map[string]string {
	requiredFields, optionalFields := configFieldsForKind(kind)
	result := make(map[string]string)
	missingVals := []string{}
	for _, fieldName := range requiredFields {
		envVarName := strings.ToUpper(kind + "_" + fieldName)
		val, found := os.LookupEnv(envVarName)
		if !found {
			missingVals = append(missingVals, envVarName)
		} else {
			result[fieldName] = val
		}
	}
	for _, fieldName := range optionalFields {
		envVarName := strings.ToUpper(kind + "_" + fieldName)
		if val, found := os.LookupEnv(envVarName); found {
			result[fieldName] = val
		}
	}
	if len(missingVals) > 0 {
		log.Warn().Str("kind", kind).
			Strs("vars", missingVals).
			Msgf("missing env vars")
	}
	return result
}
