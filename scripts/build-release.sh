#!/bin/bash
# Build release binaries for all platforms

set -e

VERSION="${1:-v1.0.0}"

echo "🏗️  Building Vectra Guard $VERSION"
echo "=================================="
echo ""

# Create dist directory
rm -rf dist
mkdir -p dist

# Platforms to build
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

echo "Building for platforms:"
for platform in "${PLATFORMS[@]}"; do
    echo "  - $platform"
done
echo ""

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    OUTPUT="dist/vectra-guard-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi
    
    echo "📦 Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$OUTPUT" main.go
    
    # Calculate checksum
    if command -v shasum &> /dev/null; then
        shasum -a 256 "$OUTPUT" | tee -a dist/checksums.txt
    fi
done

echo ""
echo "✅ Build complete!"
echo ""

# Create archives for GitHub Release
echo "📦 Creating release archives..."
cd dist
for binary in vectra-guard-*; do
    # Skip if already an archive or checksums
    if [[ "$binary" == *.tar.gz ]] || [[ "$binary" == *.zip ]] || [[ "$binary" == checksums.txt ]]; then
        continue
    fi
    
    if [[ "$binary" == *.exe ]]; then
        # Windows: create zip
        ARCHIVE="${binary}.zip"
        echo "   Creating ${ARCHIVE}..."
        zip -q "$ARCHIVE" "$binary"
    else
        # Unix: create gz (gzip only, no tar)
        ARCHIVE="${binary}.gz"
        echo "   Creating ${ARCHIVE}..."
        gzip -c "$binary" > "$ARCHIVE"
    fi
done
cd ..

# Create checksums for archives
echo ""
echo "🔐 Generating checksums..."
cd dist
shasum -a 256 *.gz *.zip > checksums-archives.txt 2>/dev/null || true
cd ..

echo ""
echo "✅ Release packages ready!"
echo ""
echo "📦 Archives in dist/:"
ls -lh dist/*.gz dist/*.zip 2>/dev/null
echo ""
echo "📝 Next steps:"
echo "   1. Create GitHub release: https://github.com/xadnavyaai/vectra-guard/releases/new"
echo "   2. Upload these files from dist/ folder:"
echo "      • vectra-guard-darwin-amd64.gz"
echo "      • vectra-guard-darwin-arm64.gz"
echo "      • vectra-guard-linux-amd64.gz"
echo "      • vectra-guard-linux-arm64.gz"
echo "      • vectra-guard-windows-amd64.exe.zip"
echo "      • checksums-archives.txt (rename to checksums.txt)"
echo "   3. Publish release"
echo ""

