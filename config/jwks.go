package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/configurator"
	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/models"
)

//go:embed jwks.yaml
var jwksFile []byte

type JSONKeyConfig struct {
	Alg jwa.Alg `yaml:"alg"`
	Key struct {
		TTL      time.Duration `yaml:"ttl"`
		Rotation time.Duration `yaml:"rotation"`
		Cache    time.Duration `yaml:"cache"`
	} `yaml:"key"`
	Token struct {
		TTL      time.Duration `yaml:"ttl"`
		Issuer   string        `yaml:"issuer"`
		Audience string        `yaml:"audience"`
		Subject  string        `yaml:"subject"`
		Leeway   time.Duration `yaml:"leeway"`
	} `yaml:"token"`
}

type KeysType map[models.KeyUsage]*JSONKeyConfig

var Keys = configurator.NewLoader[KeysType](Loader).MustLoad(
	configurator.NewConfig("", jwksFile),
)
