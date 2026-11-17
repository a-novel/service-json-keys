// Package lib protects private key material at rest. It parses the master key, and encrypts and
// decrypts key material with it.
//
// The package is tooling rather than a layer, so its functions open no span. The core operation
// that calls one reports its errors on its own span.
package lib
