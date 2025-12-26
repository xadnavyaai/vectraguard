#!/bin/bash
# Extended Test Suite for Vectra Guard
# Runs the destructive suite with additional coverage and extended branding.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export EXTENDED_MODE=true

exec "$SCRIPT_DIR/test-destructive.sh" "$@"
