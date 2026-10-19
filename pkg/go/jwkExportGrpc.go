package servicejsonkeys

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/v2/jwa"

	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

// A jwkExportGrpc adapts a [BaseClient] to the key source [NewClaimsVerifier] builds, fetching
// the public keys of a usage through the JwkList RPC.
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

	return lo.Map(res.GetKeys(), func(key *jsonkeysv2.Jwk, _ int) *jwa.JWK { return jwkFromProto(key) }), nil
}

// jwkFromProto converts a protobuf key into its jwt form.
func jwkFromProto(key *jsonkeysv2.Jwk) *jwa.JWK {
	return &jwa.JWK{
		JWKCommon: jwa.JWKCommon{
			KTY:    jwa.KTY(key.GetKty()),
			Use:    jwa.Use(key.GetUse()),
			KeyOps: lo.Map(key.GetKeyOps(), func(op string, _ int) jwa.KeyOp { return jwa.KeyOp(op) }),
			Alg:    jwa.Alg(key.GetAlg()),
			KID:    key.GetKid(),
		},
		Payload: key.GetPayload(),
	}
}
