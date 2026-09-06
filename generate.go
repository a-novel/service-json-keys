package servicejsonkeys

// Each tool is pinned in its own modfile, so that building it never pulls its
// dependency graph into this module's.

// Generate mocks.
//go:generate go tool -modfile=mockery.mod mockery

// Generate proto stubs.
//go:generate rm -rf internal/handlers/protogen
//go:generate go tool -modfile=buf.mod buf generate
