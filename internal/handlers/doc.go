// Package handlers is the transport layer of the JSON-keys application, sitting above the
// core layer. Handlers translate incoming requests into core calls and map the results back
// to protocol-level responses; business logic lives in core.
//
// Two transport implementations serve distinct audiences:
//
//   - gRPC handlers: the private API for authenticated service-to-service calls, covering
//     token signing, key retrieval, and dependency health checks. Wired by cmd/grpc.
//
//   - REST handlers: the public, unauthenticated read-only API exposing the JSON Web Keys
//     used for token verification. Wired by cmd/rest.
package handlers
