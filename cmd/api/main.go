package main

import (
	"context"
	"log"

	"github.com/a-novel/service-json-keys/models/config"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func main() {
	err := cmdpkg.App(context.Background(), config.AppPresetDefault)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}
}
