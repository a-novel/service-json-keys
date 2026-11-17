package configtest

import "os"

// GrpcURL is the address of the gRPC server the Go client tests dial, in the form <host>:<port>.
var GrpcURL = os.Getenv("GRPC_URL")
