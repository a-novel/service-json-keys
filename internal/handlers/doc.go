// Package handlers is the transport layer of the JSON-keys application, sitting above
// the service layer.
//
// There are two transport implementations, each serving a distinct audience:
//
//   - gRPC handlers: the private API for authenticated service-to-service calls. They
//     cover token signing, key retrieval, and health checks. Wired by cmd/grpc.
//
//   - HTTP handlers: the public read-only API for token verification. They expose
//     JSON Web Key endpoints without requiring authentication. Wired by cmd/rest.
//
// All handlers translate incoming requests into service-layer calls and map the results
// back to protocol-level responses. They do not contain business logic.
package handlers
