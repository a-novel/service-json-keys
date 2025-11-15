package authentication

// Generate mocks.
//go:generate go tool mockery

// Generate proto stubs.
//go:generate rm -rf internal/handlers/protogen
//go:generate go tool buf generate
