module github.com/colemickens/niche

go 1.15

replace github.com/graymeta/stow => github.com/colemickens/stow v0.2.7-0.20201203234909-b530d1a48a82

require (
	cloud.google.com/go/storage v1.10.0 // indirect
	github.com/graymeta/stow v0.2.6
	github.com/rs/zerolog v1.20.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/ulikunitz/xz v0.5.8
	github.com/urfave/cli/v2 v2.3.0
	go.mozilla.org/sops/v3 v3.6.1
	gopkg.in/yaml.v2 v2.4.0
)
