package pkg

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/internal/services"
)

type KeyUsage = string

const (
	KeyUsageAuth        KeyUsage = "auth"
	KeyUsageAuthRefresh KeyUsage = "auth-refresh"
)

type VerifyClaimsOptions struct {
	IgnoreExpired bool
}

type VerifyClaimsRequest struct {
	Usage       KeyUsage
	AccessToken string
	Options     *VerifyClaimsOptions
}

type ClaimsVerifier[C any] interface {
	VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error)
}

type claimsVerifier[C any] struct {
	service *services.ClaimsVerify[C]
}

func NewClaimsVerifier[C any](c Client) ClaimsVerifier[C] {
	service := services.NewClaimsVerify[C](c.Recipients(), c.Keys())

	return &claimsVerifier[C]{service: service}
}

func (verifier *claimsVerifier[C]) VerifyClaims(ctx context.Context, req *VerifyClaimsRequest) (*C, error) {
	return verifier.service.Exec(ctx, &services.ClaimsVerifyRequest{
		Token:         req.AccessToken,
		Usage:         req.Usage,
		IgnoreExpired: lo.FromPtr(req.Options).IgnoreExpired,
	})
}
