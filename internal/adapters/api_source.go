package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
	"github.com/a-novel/service-json-keys/models"
)

type PublicKeySourcesAPI struct {
	client *codegen.Client
}

func NewPublicKeySourcesAPI(client *codegen.Client) *PublicKeySourcesAPI {
	return &PublicKeySourcesAPI{client: client}
}

func (api *PublicKeySourcesAPI) SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*jwa.JWK, error) {
	rawRes, err := api.client.ListPublicKeys(ctx, codegen.ListPublicKeysParams{Usage: codegen.KeyUsage(usage)})
	if err != nil {
		return nil, err
	}

	res, ok := rawRes.(*codegen.ListPublicKeysOKApplicationJSON)
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
