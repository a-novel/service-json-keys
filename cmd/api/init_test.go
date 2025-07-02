package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/a-novel-kit/configurator/chans"
	"github.com/a-novel-kit/configurator/utilstest"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/pkg"
)

var logs *chans.MultiChan[string]

func _patchSTD() {
	patchedStd, _, err := utilstest.MonkeyPatchStderr()
	if err != nil {
		panic(err)
	}

	logs, _, err = utilstest.CaptureSTD(patchedStd)
	if err != nil {
		panic(err)
	}

	go func() {
		listener := logs.Register()
		for msg := range listener {
			// Forward logs to default system outputs, in case we need them for debugging.
			log.Println("forwarded:", msg)
		}
	}()
}

func _runKeysRotation() {
	// Run keys rotation.
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rotateKeysPath := path.Join(filepath.Dir(pwd), "..", "cmd", "rotatekeys", "main.go")

	out, err := exec.CommandContext(context.Background(), "go", "run", rotateKeysPath).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("rotate keys: %v\n%v", err, string(out)))
	}
}

// Create a separate database to run integration tests.
func init() {
	_patchSTD()

	_runKeysRotation()

	go func() {
		main()
	}()

	_, err := pkg.NewAPIClient(context.Background(), fmt.Sprintf("http://127.0.0.1:%v/v1", config.API.Port))
	if err != nil {
		panic(err)
	}
}
