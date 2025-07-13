#!/bin/bash

set -e

go run cmd/migrations/main.go
go run cmd/rotatekeys/main.go
