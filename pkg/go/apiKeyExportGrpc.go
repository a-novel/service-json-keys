package servicejsonkeys

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel/service-json-keys/v2/pkg/go/internal/tokens"
)

// An apiKeyExportGrpc adapts an [APIKeyReader] to the key source [NewAPIKeyVerifier] builds,
// fetching an API key through the APIKeyGet RPC.
type apiKeyExportGrpc struct {
	reader APIKeyReader
}

func newAPIKeyExportGrpc(reader APIKeyReader) *apiKeyExportGrpc {
	return &apiKeyExportGrpc{reader: reader}
}

func (api *apiKeyExportGrpc) SelectKey(ctx context.Context, id string) (*tokens.APIKey, error) {
	res, err := api.reader.APIKeyGet(ctx, &APIKeyGetRequest{Id: id})
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: %w", tokens.ErrAPIKeyNotFound, err)
	}

	if err != nil {
		return nil, err
	}

	key := res.GetApiKey()

	return &tokens.APIKey{
		ID:          key.GetId(),
		Purpose:     key.GetPurpose(),
		Key:         jwkFromProto(key.GetJwk()),
		Deactivated: key.GetDeactivated(),
		Leaked:      key.GetLeaked(),
	}, nil
}
