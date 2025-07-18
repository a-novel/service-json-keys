package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel/golib/config"

	"github.com/a-novel/service-json-keys/models"
)

//go:embed jwks.config.yaml
var defaultJWKSConfigFile []byte

var JWKSPresetDefault = config.MustUnmarshal[map[models.KeyUsage]*JWKS](yaml.Unmarshal, defaultJWKSConfigFile)
