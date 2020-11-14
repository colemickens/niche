package main

import "fmt"

func getInitialStorageConfigMap(kind string) (map[string]string, error) {
	switch kind {
	case "azure":
		return map[string]string{
			"account": kind + "_ACCOUNT_NAME_HERE",
			"key":     kind + "_ACCOUNT_ACCESS_KEY_HERE",
		}, nil
	case "aws":
		return map[string]string{
			"auth_type":     kind + "_ACCOUNT_NAME_HERE",
			"access_key_id": kind + "_ACCOUNT_ACCESS_KEY_HERE",
			"secret_key":    kind + "_ACCOUNT_ACCESS_KEY_HERE",
			"region":        kind + "_ACCOUNT_ACCESS_KEY_HERE",
			"endpoint":      kind + "_ACCOUNT_ACCESS_KEY_HERE",
			"disable_ssl":   kind + "_ACCOUNT_ACCESS_KEY_HERE",
			"v2_signing":    kind + "",
		}, nil
	}

	return nil, fmt.Errorf("unsupported kind for init", kind)
}
