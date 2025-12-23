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
        # Unix: create gz (gzip only)
        ARCHIVE="${binary}.gz"
        echo "📦 Creating ${ARCHIVE}..."
        gzip -c "$binary" > "$ARCHIVE"
    fi
done

echo ""
echo "✅ Packaging complete!"
echo ""
echo "📦 Archives ready for GitHub:"
ls -lh *.gz *.zip 2>/dev/null
echo ""
echo "🚀 Upload these files to GitHub Release:"
echo "   • vectra-guard-darwin-amd64.gz"
echo "   • vectra-guard-darwin-arm64.gz"
echo "   • vectra-guard-linux-amd64.gz"
echo "   • vectra-guard-linux-arm64.gz"
echo "   • vectra-guard-windows-amd64.exe.zip"
echo "   • checksums.txt"
echo ""

