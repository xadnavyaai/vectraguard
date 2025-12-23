#!/bin/bash
# Package binaries for GitHub Release
# Creates tar.gz archives that GitHub accepts

set -e

echo "📦 Packaging binaries for GitHub Release..."
echo ""

cd dist

# Package each binary
for binary in vectra-guard-*; do
    # Skip if already an archive or checksums
    if [[ "$binary" == *.tar.gz ]] || [[ "$binary" == *.zip ]] || [[ "$binary" == checksums.txt ]]; then
        continue
    fi
    
    if [[ "$binary" == *.exe ]]; then
        # Windows: create zip
        ARCHIVE="${binary}.zip"
        echo "📦 Creating ${ARCHIVE}..."
        zip -q "$ARCHIVE" "$binary"
    else
        # Unix: create tar.gz
        ARCHIVE="${binary}.tar.gz"
        echo "📦 Creating ${ARCHIVE}..."
        tar czf "$ARCHIVE" "$binary"
    fi
done

echo ""
echo "✅ Packaging complete!"
echo ""
echo "📦 Archives ready for GitHub:"
ls -lh *.tar.gz *.zip 2>/dev/null
echo ""
echo "🚀 Upload these files to GitHub Release:"
echo "   • vectra-guard-darwin-amd64.tar.gz"
echo "   • vectra-guard-darwin-arm64.tar.gz"
echo "   • vectra-guard-linux-amd64.tar.gz"
echo "   • vectra-guard-linux-arm64.tar.gz"
echo "   • vectra-guard-windows-amd64.exe.zip"
echo "   • checksums.txt"
echo ""

