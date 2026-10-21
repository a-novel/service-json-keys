// Package tokens verifies the tokens the service signs, for the Go client. It builds cached
// verification plugins from each usage's public keys, then checks a token's signature and claims
// against the usage configuration.
//
// The service only signs, so verification lives with its one consumer. The client therefore links
// no service layer.
package tokens
