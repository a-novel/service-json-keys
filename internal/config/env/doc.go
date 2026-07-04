// Package env reads the service configuration from environment variables and exposes
// each setting as a package-level variable, resolved once at package initialization.
// Settings that have a fallback apply their Default value when the variable is unset
// or empty; the rest are used as read.
package env
