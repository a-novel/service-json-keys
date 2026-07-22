package handlers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/a-novel-kit/jwt/v2/jwa"
)

// ErrPrivateKeyMaterial is returned when a key bound for the public API carries a
// secret member. It is a server fault, not a client one: the caller asked for
// something legitimate and a layer below produced material this API may not serve.
var ErrPrivateKeyMaterial = errors.New("refusing to serve private key material")

// privateJWKMembers are the JWK parameters that carry secret material: the
// symmetric key octets (RFC 7518 §6.4) and the RSA and EC private components
// (§6.3.2, §6.2.2).
var privateJWKMembers = []string{"d", "k", "p", "q", "dp", "dq", "qi", "oth"}

// assertPublic reports [ErrPrivateKeyMaterial] when any of jwks carries a secret
// member.
//
// The HTTP API is public, so this is the last point at which a mistake in a layer
// below is still inside the process. Extraction already refuses to produce private
// material without authorization; this holds if that ever stops being true, and it
// holds for material that reaches a handler by some route nobody has thought of.
//
// A leak here is unauthenticated and permanent — a key that reached the internet is
// compromised whatever happens next — which is what buys a check this cheap its
// place on the hot path.
func assertPublic(jwks ...*jwa.JWK) error {
	for _, jwk := range jwks {
		if jwk == nil || len(jwk.Payload) == 0 {
			continue
		}

		var members map[string]json.RawMessage

		// Unparseable payload: report it rather than pass it on. Serving bytes this
		// package cannot read to a public endpoint is the position this guard exists
		// to refuse.
		err := json.Unmarshal(jwk.Payload, &members)
		if err != nil {
			return fmt.Errorf("%w: unreadable payload for key %s", ErrPrivateKeyMaterial, jwk.KID)
		}

		for _, member := range privateJWKMembers {
			if _, found := members[member]; found {
				return fmt.Errorf("%w: key %s carries %q", ErrPrivateKeyMaterial, jwk.KID, member)
			}
		}
	}

	return nil
}
