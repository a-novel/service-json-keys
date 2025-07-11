package models

import (
	_ "embed"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/a-novel/golib/config"

	"github.com/a-novel-kit/jwt/jwa"
)

type JSONKeyConfig struct {
	Alg jwa.Alg `json:"alg" yaml:"alg"`
	Key struct {
		TTL      time.Duration `json:"ttl"      yaml:"ttl"`
		Rotation time.Duration `json:"rotation" yaml:"rotation"`
		Cache    time.Duration `json:"cache"    yaml:"cache"`
	} `json:"key" yaml:"key"`
	Token struct {
		TTL      time.Duration `json:"ttl"      yaml:"ttl"`
		Issuer   string        `json:"issuer"   yaml:"issuer"`
		Audience string        `json:"audience" yaml:"audience"`
		Subject  string        `json:"subject"  yaml:"subject"`
		Leeway   time.Duration `json:"leeway"   yaml:"leeway"`
	} `json:"token" yaml:"token"`
}

//go:embed jwks.yaml
var defaultJWKSConfigFile []byte

var DefaultJWKSConfig = config.MustUnmarshal[map[KeyUsage]*JSONKeyConfig](yaml.Unmarshal, defaultJWKSConfigFile)
