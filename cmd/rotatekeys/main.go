package main

import (
	"context"
	"log"

	"github.com/a-novel/service-json-keys/models/config"
	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func main() {
	err := cmdpkg.JobRotateKeys(context.Background(), config.JobRotateKeysPresetDefault)
	if err != nil {
		log.Fatalf("failed to run job: %v", err)
	}
}
