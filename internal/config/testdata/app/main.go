package main

import (
	"fmt"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

func main() {
	fmt.Println(config.AppPresetDefault.Rest.Timeouts.Read)
}
