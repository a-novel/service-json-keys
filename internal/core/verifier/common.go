package verifier

import "errors"

// ErrConfigNotFound is returned when no JWK configuration is registered for a key usage.
var ErrConfigNotFound = errors.New("no config found for the requested usage")

// ErrPresetUnknown is returned when an algorithm has no corresponding preset entry.
var ErrPresetUnknown = errors.New("unknown jwk preset")

// ErrPresetUnknownAlgorithm is returned when a key configuration selects an unsupported algorithm.
var ErrPresetUnknownAlgorithm = errors.New("unknown jwk algorithm")
