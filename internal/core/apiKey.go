package core

// APIKey is an API key as the service serves it: a public key made for one purpose, with its
// states. It never names the owner of the key.
type APIKey struct {
	// ID names the key. A token the key signed carries it in its "kid" header.
	ID string
	// Purpose is the one thing the key may do.
	Purpose string
	// Key is the public key, whose KID is the ID.
	Key *Jwk
	// Deactivated is whether the key no longer authenticates a request. What it signed before still
	// verifies.
	Deactivated bool
	// Leaked is whether the private key leaked, so nothing the key signed is trusted. A leaked system
	// API key leaves its setting instead, so none carries it.
	Leaked bool
}
