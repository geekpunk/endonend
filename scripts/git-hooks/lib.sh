#!/usr/bin/env bash
# Shared helpers for the Go pre-commit / pre-push hooks. Not a hook itself.
set -euo pipefail

find_go_modules() {
    find . -name go.mod -not -path '*/vendor/*' -exec dirname {} \; | sort -u
}

module_for_file() {
    local dir
    dir="$(dirname "$1")"
    while [ "$dir" != "." ] && [ ! -f "$dir/go.mod" ]; do
        dir="$(dirname "$dir")"
    done
    [ -f "$dir/go.mod" ] && echo "$dir"
}

require_golangci_lint() {
    if ! command -v golangci-lint >/dev/null 2>&1; then
        echo "$1: golangci-lint not found. Install it (e.g. 'brew install golangci-lint') and retry." >&2
        exit 1
    fi
}

check_go_modules() {
    local hook_name="$1"
    shift
    local modules=("$@")
    local status=0

    for module in "${modules[@]}"; do
        echo "$hook_name: checking Go module $module"
        if ! (cd "$module" && go test ./...); then
            status=1
        fi
        if ! (cd "$module" && golangci-lint run ./...); then
            status=1
        fi
    done

    if [ "$status" -ne 0 ]; then
        echo "$hook_name: Go tests or lint failed; blocked." >&2
        exit 1
    fi
}
