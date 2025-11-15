package pkg

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
)

// jwkExportGrpc is a service used to wrap an api client so it can be used as a
// cached jwk.Source to reduce load on the network.
//
// This exporter is meant for external usage.
type jwkExportGrpc struct {
	client BaseClient
}

func newJwkExportGrpc(client BaseClient) *jwkExportGrpc {
	return &jwkExportGrpc{client: client}
}

func (api *jwkExportGrpc) SearchKeys(ctx context.Context, usage string) ([]*jwa.JWK, error) {
	res, err := api.client.JwkList(ctx, &JwkListRequest{Usage: usage})
	if err != nil {
		return nil, err
	}

	keys := lo.Map(res.GetKeys(), func(item *protogen.Jwk, index int) *jwa.JWK {
		return &jwa.JWK{
			JWKCommon: jwa.JWKCommon{
				KTY: jwa.KTY(item.GetKty()),
				Use: jwa.Use(item.GetUse()),
				KeyOps: lo.Map(item.GetKeyOps(), func(item string, index int) jwa.KeyOp {
					return jwa.KeyOp(item)
				}),
				Alg: jwa.Alg(item.GetAlg()),
				KID: item.GetKid(),
			},
			Payload: item.GetPayload(),
		}
	})

	return keys, nil
}
