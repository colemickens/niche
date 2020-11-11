package nixclient

type NixPathInfo struct {
	Path       string `json:"path"`
	NarHash    string
	NarSize    int64
	References []string
	Deriver    string
	Signatures []string
}

// TODO: phase out! (libnixstore; niche-RIIR (after nix-riir, etc); etc...)
type NixClient interface {
	PathInfo(storePath string) (pathInfo *NixPathInfo, err error)
	QueryPaths(storePath string) (dependencies []string, err error)
	Build(thing string, extraArgs ...string) (string, error)
}
