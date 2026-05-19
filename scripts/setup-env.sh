#!/bin/bash
# Sets up environment variables for local development and testing. Each variable uses the
# assign-if-unset pattern (${VAR:=default}), so pre-exported values are preserved and only
# missing ones are filled in. Source this file before running any local service or test command.

# Dummy master key for local use only — never use this value in production or any shared environment.
export APP_MASTER_KEY="${APP_MASTER_KEY:="fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"}"
