#!/bin/bash
# Wrapper for vectra-guard sandbox deps installer
set -e

VG_BIN="${VG_BIN:-vectra-guard}"

if ! command -v "${VG_BIN}" &> /dev/null; then
    if [ -x "./vectra-guard" ]; then
        VG_BIN="./vectra-guard"
    else
        echo "❌ vectra-guard not found."
        echo "   Install it first:"
        echo "   curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash"
        exit 1
    fi
fi

exec "${VG_BIN}" sandbox deps install "$@"
