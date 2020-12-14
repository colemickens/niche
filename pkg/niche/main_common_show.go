package niche

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
)

func show(cacheNameRaw string) error {
	cacheURLStr := cacheNameRaw
	cacheURL, err := preprocessHostArg(cacheURLStr)
	if err != nil {
		return err
	}

	u := cacheURL
	u.Path = path.Join(u.Path, wkPublicConfig)

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	var pc publicNicheConfig
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	err = dec.Decode(&pc)
	if err != nil {
		return err
	}
	publicKey := string(pc.PublicKey)
	fmt.Println(publicKey)
	return nil
}
