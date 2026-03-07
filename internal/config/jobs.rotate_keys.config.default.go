package config

import (
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

var JobRotateKeysPresetDefault = JobRotateKeys{
	App: Main{
		Name:      env.AppName + "-job-rotate-keys",
		MasterKey: env.AppMasterKey,
	},
	Jwk: JwkPresetDefault,

	Otel: lo.If[otel.Config](!env.Otel, &otelpresets.Disabled{}).
		ElseIf(env.GcloudProjectId == "", &otelpresets.Local{
			FlushTimeout: OtelFlushTimeout,
		}).
		Else(&otelpresets.Gcloud{
			ProjectID:    env.GcloudProjectId,
			FlushTimeout: OtelFlushTimeout,
		}),
	Postgres: PostgresPresetDefault,
}
