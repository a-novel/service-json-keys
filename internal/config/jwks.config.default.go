package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel/golib/config"
)

//go:embed jwks.config.yaml
var defaultJWKSConfigFile []byte

var JwkPresetDefault = config.MustUnmarshal[map[string]*Jwk](yaml.Unmarshal, defaultJWKSConfigFile)
