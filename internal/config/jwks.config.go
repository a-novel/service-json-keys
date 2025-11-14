package config

import (
	_ "embed"
	"time"

	"github.com/a-novel-kit/jwt/jwa"
)

type JwkKey struct {
	TTL      time.Duration `json:"ttl"      yaml:"ttl"`
	Rotation time.Duration `json:"rotation" yaml:"rotation"`
	Cache    time.Duration `json:"cache"    yaml:"cache"`
}

type JwkToken struct {
	TTL      time.Duration `json:"ttl"      yaml:"ttl"`
	Issuer   string        `json:"issuer"   yaml:"issuer"`
	Audience string        `json:"audience" yaml:"audience"`
	Subject  string        `json:"subject"  yaml:"subject"`
	Leeway   time.Duration `json:"leeway"   yaml:"leeway"`
}

type Jwk struct {
	Alg   jwa.Alg  `json:"alg"   yaml:"alg"`
	Key   JwkKey   `json:"key"   yaml:"key"`
	Token JwkToken `json:"token" yaml:"token"`
}
