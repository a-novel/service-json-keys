// Package services is the business logic layer of the JSON-keys application, sitting between
// the DAO layer (data access) and the handler layer (transport).
//
// It handles the full key lifecycle, JWT signing and verification, and the construction
// of typed in-memory key sources that wire per-usage signing and verification plugins.
package services
