// Package servicejsonkeys provides a gRPC client for the JSON Keys service.
//
// # How the service works
//
// The JSON Keys service is a centralized key manager. Services register named usages, each with
// its own signing algorithm, key rotation schedule and token parameters. The service holds all
// private keys and signs on behalf of callers, so private key material never leaves the server.
// Any service with access to the public keys can verify tokens locally.
//
// # Tokens and claims
//
// Tokens issued by this service are compact JWTs: three base64url-encoded segments
// (header, payload, signature) joined by dots, as defined by RFC 7519. The payload is a
// JSON object carrying the caller-supplied claims wrapped in the standard JWT claim
// envelope added by the service. Claims can be any JSON-serializable value.
//
// # Using this package
//
// [NewClient] dials the service, and [NewClaimsVerifier] builds per-usage public-key sources
// on top of it. A source fetches its keys on first use and caches them, so token verification
// needs no network call per token:
//
//	c, err := servicejsonkeys.NewClient("host:port", grpc.WithTransportCredentials(...))
//	if err != nil { ... }
//	defer c.Close()
//
//	// Verify a token locally; MyClaims must match the claims type used when signing for KeyUsageAuth.
//	verifier, err := servicejsonkeys.NewClaimsVerifier[MyClaims](c)
//	if err != nil { ... }
//	claims, err := verifier.VerifyClaims(ctx, &servicejsonkeys.VerifyClaimsRequest{
//	    Usage:       servicejsonkeys.KeyUsageAuth,
//	    AccessToken: tokenString, // compact JWT string
//	})
package servicejsonkeys
