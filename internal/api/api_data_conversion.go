package api

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
)

func (api *API) jwkToModel(src *jwa.JWK) (*codegen.JWK, error) {
	rawPayload := new(codegen.JWKAdditional)

	err := rawPayload.UnmarshalJSON(src.Payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	kid, err := uuid.Parse(src.KID)
	if err != nil {
		return nil, fmt.Errorf("parse kid: %w", err)
	}

	return &codegen.JWK{
		Kty:             codegen.KTY(src.KTY),
		Use:             codegen.Use(src.Use),
		KeyOps:          lo.Map(src.KeyOps, func(item jwa.KeyOp, _ int) codegen.KeyOp { return codegen.KeyOp(item) }),
		Alg:             codegen.Alg(src.Alg),
		Kid:             codegen.OptKID{Value: codegen.KID(kid), Set: true},
		AdditionalProps: *rawPayload,
	}, nil
}

func (api *API) jwksToModels(src ...*jwa.JWK) ([]codegen.JWK, error) {
	output := make([]codegen.JWK, len(src))

	for i, jwk := range src {
		model, err := api.jwkToModel(jwk)
		if err != nil {
			return nil, fmt.Errorf("convert jwk to model: %w", err)
		}

		output[i] = *model
	}

	return output, nil
}
