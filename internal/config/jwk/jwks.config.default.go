package jwk

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel-kit/golib/config"
)

//go:embed jwks.config.yaml
var defaultJWKSConfigFile []byte

// PresetDefault is the default JWK configuration for all registered usages,
// loaded from the bundled YAML file.
var PresetDefault = config.MustUnmarshal[map[string]*Jwk](yaml.Unmarshal, defaultJWKSConfigFile)
