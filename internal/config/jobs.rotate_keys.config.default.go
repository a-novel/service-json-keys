package config

import (
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

var JobRotateKeysPresetDefault = JobRotateKeys{
	App: Main{
		Name:      env.AppName + "-job-rotate-keys",
		MasterKey: env.AppMasterKey,
	},
	Jwk: JwkPresetDefault,

	Otel: lo.If[otel.Config](!env.Otel, &NoOtel).
		ElseIf(env.GcloudProjectId == "", &OtelDev).
		Else(&OtelProd),
	Postgres: PostgresPresetDefault,
}
