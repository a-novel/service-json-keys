package api

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/models/api"
)

func (api *API) jwkToModel(src *jwa.JWK) (*apimodels.JWK, error) {
	rawPayload := new(apimodels.JWKAdditional)

	err := rawPayload.UnmarshalJSON(src.Payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	kid, err := uuid.Parse(src.KID)
	if err != nil {
		return nil, fmt.Errorf("parse kid: %w", err)
	}

	return &apimodels.JWK{
		Kty:             apimodels.KTY(src.KTY),
		Use:             apimodels.Use(src.Use),
		KeyOps:          lo.Map(src.KeyOps, func(item jwa.KeyOp, _ int) apimodels.KeyOp { return apimodels.KeyOp(item) }),
		Alg:             apimodels.Alg(src.Alg),
		Kid:             apimodels.OptKID{Value: apimodels.KID(kid), Set: true},
		AdditionalProps: *rawPayload,
	}, nil
}

func (api *API) jwksToModels(src ...*jwa.JWK) ([]apimodels.JWK, error) {
	output := make([]apimodels.JWK, len(src))

	for i, jwk := range src {
		model, err := api.jwkToModel(jwk)
		if err != nil {
			return nil, fmt.Errorf("convert jwk to model: %w", err)
		}

		output[i] = *model
	}

	return output, nil
}
