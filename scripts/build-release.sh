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
echo "📦 Binaries in dist/:"
ls -lh dist/
echo ""
echo "📝 Next steps:"
echo "   1. Test binaries"
echo "   2. Create GitHub release: https://github.com/xadnavyaai/vectra-guard/releases/new"
echo "   3. Upload binaries from dist/ folder"
echo "   4. Publish release"
echo ""

