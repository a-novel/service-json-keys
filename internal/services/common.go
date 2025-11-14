package services

import "errors"

var ErrConfigNotFound = errors.New("no config found for the requested usage")
