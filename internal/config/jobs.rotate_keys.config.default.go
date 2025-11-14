package config

import (
	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-json-keys/internal/config/env"
)

const (
	JobsRotateKeysName = "service-json-keys-job-rotate-keys"
)

var JobRotateKeysPresetDefault = JobRotateKeys{
	App: Main{
		Name:      config.LoadEnv(env.AppName, JobsRotateKeysName, config.StringParser),
		MasterKey: env.AppMasterKey,
	},
	Jwk: JwkPresetDefault,

	Otel: lo.If[otel.Config](!env.Otel, &NoOtel).
		ElseIf(env.GcloudProjectId == "", &OtelDev).
		Else(&OtelProd),
	Postgres: PostgresPresetDefault,
}
