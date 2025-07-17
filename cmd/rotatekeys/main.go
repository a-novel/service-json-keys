package main

import (
	"context"
	"log"

	cmdpkg "github.com/a-novel/service-json-keys/pkg/cmd"
)

func main() {
	err := cmdpkg.JobRotateKeys(context.Background(), cmdpkg.JobRotateKeysConfigDefault)
	if err != nil {
		log.Fatalf("failed to run job: %v", err)
	}
}
