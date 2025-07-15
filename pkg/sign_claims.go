package pkg

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/api"
)

type ClaimsSigner struct {
	client *apimodels.Client
}

func NewClaimsSigner(client *apimodels.Client) *ClaimsSigner {
	return &ClaimsSigner{client: client}
}

func (pkg *ClaimsSigner) SignClaims(ctx context.Context, usage models.KeyUsage, claims any) (string, error) {
	serialized, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	req := apimodels.SignClaimsReq{}

	err = req.UnmarshalJSON(serialized)
	if err != nil {
		return "", fmt.Errorf("unmarshal claims: %w", err)
	}

	rawRes, err := pkg.client.SignClaims(ctx, req, apimodels.SignClaimsParams{Usage: apimodels.KeyUsage(usage)})
	if err != nil {
		return "", fmt.Errorf("sign claims: %w", err)
	}

	res, ok := rawRes.(*apimodels.Token)
	if !ok {
		return "", fmt.Errorf("unexpected response type: %T", rawRes)
	}

	return res.Token, nil
}
