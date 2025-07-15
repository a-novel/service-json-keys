package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/api"
)

type PublicKeySourcesAPI struct {
	client *apimodels.Client
}

func NewPublicKeySourcesAPI(client *apimodels.Client) *PublicKeySourcesAPI {
	return &PublicKeySourcesAPI{client: client}
}

func (api *PublicKeySourcesAPI) SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*jwa.JWK, error) {
	rawRes, err := api.client.ListPublicKeys(ctx, apimodels.ListPublicKeysParams{Usage: apimodels.KeyUsage(usage)})
	if err != nil {
		return nil, err
	}

	res, ok := rawRes.(*apimodels.ListPublicKeysOKApplicationJSON)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", rawRes)
	}

	keys := make([]*jwa.JWK, len(*res))

	for i, keyModel := range *res {
		serialized, err := keyModel.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshal key model: %w", err)
		}

		err = json.Unmarshal(serialized, &keys[i])
		if err != nil {
			return nil, fmt.Errorf("unmarshal key model: %w", err)
		}
	}

	return keys, nil
}
