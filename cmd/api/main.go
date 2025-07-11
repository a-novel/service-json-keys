package main

import (
	"context"
	"log"

	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func main() {
	err := cmdpkg.App(context.Background(), cmdpkg.AppConfigDefault)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}
}
