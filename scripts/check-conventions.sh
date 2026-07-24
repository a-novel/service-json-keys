#!/usr/bin/env bash
#
# Filename and layout conventions from write-go and write-go-service.
#
# These cannot be semgrep rules: semgrep's `paths` only *filters* which files a
# pattern runs against, so it can never express "flag a file whose NAME does not
# match a pattern". Everything here keys on the path, not the contents.
#
# Exemptions are deliberate and explicit: add the path to the matching ALLOW_*
# list below rather than loosening a pattern, so every exception stays visible.

set -euo pipefail

fail=0

# Report one violation and mark the run failed.
report() {
	local file="$1" rule="$2" why="$3"
	printf '::error file=%s::[%s] %s\n' "$file" "$rule" "$why"
	fail=1
}

# Production Go files, excluding generated trees and tests.
go_sources() {
	find internal cmd pkg -name '*.go' \
		-not -name '*_test.go' \
		-not -path '*/mocks/*' \
		-not -path '*/protogen/*' \
		-not -path '*/gen/*' \
		2>/dev/null | sort
}

# --- camelCase file names -----------------------------------------------------
# write-go: multi-word file names are camelCase — never snake_case.
# The layer prefixes below are dot-separated, so a dot is legal; an underscore
# in a non-test file never is.
while IFS= read -r f; do
	[ -z "$f" ] && continue
	base="$(basename "$f")"
	case "$base" in
	*_*) report "$f" "file-camelcase" "snake_case file name; use camelCase (write-go)" ;;
	esac
done < <(go_sources)

# --- layer prefixes -----------------------------------------------------------
# write-go-service fixes the prefix per layer. A file that does not carry its
# layer's prefix is either misplaced or misnamed, and both matter for discovery.
#
# Only files sitting DIRECTLY in a layer are checked: internal/config/env and
# internal/config/configtest are their own packages, not config subjects, and the
# prefix rule does not describe them. `doc.go` is the package doc and is exempt
# everywhere.
while IFS= read -r f; do
	[ -z "$f" ] && continue
	base="$(basename "$f")"
	[ "$base" = "doc.go" ] && continue
	# skip anything nested deeper than the layer directory itself
	[ "$(dirname "$f" | awk -F/ '{print NF}')" -gt 2 ] && continue
	case "$f" in
	internal/dao/*)
		case "$base" in
		pg.*) ;;
		*) report "$f" "layer-prefix" "dao files are pg.<entity>[<Operation>].go" ;;
		esac
		;;
	internal/handlers/*)
		case "$base" in
		rest.* | grpc.*) ;;
		http.*) report "$f" "legacy-http-prefix" "handlers use rest.*, never http.* (write-go-service)" ;;
		*) report "$f" "layer-prefix" "handler files are <rest|grpc>.<entity><Operation>.go" ;;
		esac
		;;
	internal/config/*)
		case "$base" in
		*.config.go | *.config.default.go) ;;
		*) report "$f" "layer-prefix" "config files are <subject>.config[.default].go" ;;
		esac
		;;
	esac
done < <(go_sources)

# --- tests mirror their production file ---------------------------------------
# write-go: a test file mirrors the production file it covers. A test whose
# sibling does not exist is usually a rename that left the test behind.
ALLOW_ORPHAN_TESTS=(
	"internal/core/claims_test.go" # covers the claims round-trip across sign+verify
	"internal/core/utils_test.go"  # shared core_test fixtures, covers no single file
)
while IFS= read -r t; do
	[ -z "$t" ] && continue
	src="${t%_test.go}.go"
	[ -f "$src" ] && continue
	skip=0
	for allowed in "${ALLOW_ORPHAN_TESTS[@]}"; do
		[ "$t" = "$allowed" ] && skip=1 && break
	done
	[ "$skip" = "1" ] && continue
	report "$t" "test-mirrors-source" "no $src; a test file mirrors the file it covers (write-go)"
done < <(find internal pkg -name '*_test.go' -not -path '*/mocks/*' -not -path '*/protogen/*' 2>/dev/null | sort)

if [ "$fail" = "0" ]; then
	echo "check-conventions: clean"
fi

exit "$fail"
