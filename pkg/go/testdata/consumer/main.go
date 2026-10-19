// Command consumer imports the client the way another module does, to prove the import ignores
// the server's environment.
package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

func main() {
	request := &servicejsonkeys.JwkListRequest{Usage: servicejsonkeys.KeyUsageAuth}
	if request.GetUsage() != servicejsonkeys.KeyUsageAuth {
		panic("client request alias lost its generated methods")
	}

	client, err := servicejsonkeys.NewClient("localhost:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if _, err := servicejsonkeys.NewClaimsVerifier[map[string]any](client); err != nil {
		panic(err)
	}
}
