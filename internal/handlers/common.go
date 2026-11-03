package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// sendPublicKeys answers a read of public keys as JSON a cache can revalidate. The body carries a
// strong ETag, its SHA-256 (RFC 9110 §8.8.3), and [http.ServeContent] answers an If-None-Match that
// matches it with 304 Not Modified (RFC 9110 §13.1.2, §15.4.5), keeping the ETag and Cache-Control.
//
// Cache-Control is no-cache (RFC 9111 §5.2.2.4): a cache stores the keys but revalidates them on
// every use, and gives them no freshness lifetime. A rotated key signs as soon as the signer's cache
// takes it, and a verifier that meets its unknown kid refetches the set (OpenID Connect Core §10.1.1).
// A stored copy served without revalidation would then fail that verification. A revoked key leaves
// the set on the next revalidation too.
//
// The ETag has to be known before the headers go out, so the body is encoded whole first.
func sendPublicKeys(w http.ResponseWriter, r *http.Request, keys any) error {
	body, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("encode keys: %w", err)
	}

	digest := sha256.Sum256(body)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Cache-Control", "no-cache")
	header.Set("ETag", `"`+base64.RawURLEncoding.EncodeToString(digest[:])+`"`)

	// The keys carry no modification time, so the ETag alone decides.
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))

	return nil
}
