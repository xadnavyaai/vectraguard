# Packaging Action Plan

## ✅ COMPLETED

- [x] Created install script (`install.sh`)
- [x] Created build script (`scripts/build-release.sh`)
- [x] Built binaries for all platforms
- [x] Created Homebrew formula (`homebrew/vectra-guard.rb`)
- [x] Generated checksums

## 🎯 NEXT STEPS (Do These Now)

### Step 1: Upload Binaries to GitHub Release (5 minutes)

1. Go to: https://github.com/xadnavyaai/vectra-guard/releases/tag/v1.0.0
2. Click **"Edit"**
3. Upload all files from `dist/` folder:
   - `vectra-guard-darwin-amd64`
   - `vectra-guard-darwin-arm64`
   - `vectra-guard-linux-amd64`
   - `vectra-guard-linux-arm64`
   - `vectra-guard-windows-amd64.exe`
   - `checksums.txt`
4. Click **"Update release"**

### Step 2: Test Install Script (2 minutes)

```bash
# Test the install script
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Expected result**: Binary downloads and installs to `/usr/local/bin/vectra-guard`

### Step 3: Create Homebrew Tap (15 minutes) - OPTIONAL

1. Create new GitHub repository: `homebrew-tap`
   - Repository name MUST be exactly: `homebrew-tap`
   - Make it public

2. Add formula:
```bash
cd homebrew-tap
mkdir Formula
cp ../vectra-guard/homebrew/vectra-guard.rb Formula/
```

3. Update SHA256 in formula:
```bash
# Create archive
cd vectra-guard
git archive --format=tar.gz --prefix=vectra-guard-1.0.0/ v1.0.0 > vectra-guard-1.0.0.tar.gz

# Get SHA256
shasum -a 256 vectra-guard-1.0.0.tar.gz
# Copy this SHA256 and paste it in the formula
```

4. Push:
```bash
cd homebrew-tap
git add Formula/vectra-guard.rb
git commit -m "Add vectra-guard formula"
git push origin main
```

5. Test:
```bash
brew tap xadnavyaai/tap
brew install vectra-guard
```

### Step 4: Push to Docker Hub (10 minutes) - OPTIONAL

```bash
# Login
docker login

# Build and tag
docker build -t xadnavyaai/vectra-guard:v1.0.0 .
docker tag xadnavyaai/vectra-guard:v1.0.0 xadnavyaai/vectra-guard:latest

# Push
docker push xadnavyaai/vectra-guard:v1.0.0
docker push xadnavyaai/vectra-guard:latest
```

## 📊 Installation Methods Available

After completing Step 1 & 2:

### ✅ Method 1: Install Script (READY NOW)
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```
**Works on**: macOS, Linux, any Unix

### ✅ Method 2: Direct Download (READY AFTER STEP 1)
```bash
# macOS ARM64 (M1/M2/M3)
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-arm64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/

# Linux
curl -L https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-linux-amd64 -o vectra-guard
chmod +x vectra-guard
sudo mv vectra-guard /usr/local/bin/
```

### ✅ Method 3: Go Install (ALREADY WORKS)
```bash
go install github.com/xadnavyaai/vectra-guard@latest
```

### ⏳ Method 4: Homebrew (AFTER STEP 3)
```bash
brew install xadnavyaai/tap/vectra-guard
```

### ⏳ Method 5: Docker (AFTER STEP 4)
```bash
docker pull xadnavyaai/vectra-guard:latest
```

## 🎯 Priority

**MUST DO** (Required for easy installation):
- ✅ Step 1: Upload binaries to GitHub Release
- ✅ Step 2: Test install script

**NICE TO HAVE** (Do later):
- ⏳ Step 3: Homebrew tap (most professional, but takes time)
- ⏳ Step 4: Docker Hub (for container users)

## 🎉 Success Criteria

After Step 1 & 2, customers can install with:

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**That's it!** One command, works everywhere. ✨

## 📝 Files Included

- ✅ `install.sh` - Universal installer
- ✅ `scripts/build-release.sh` - Build script
- ✅ `homebrew/vectra-guard.rb` - Homebrew formula
- ✅ `dist/` - Pre-built binaries (5 platforms + checksums)
- ✅ `DISTRIBUTION_GUIDE.md` - Complete documentation
- ✅ This action plan

## ⏭️ Next Action

**Upload binaries to GitHub release now!**

Go to: https://github.com/xadnavyaai/vectra-guard/releases/tag/v1.0.0

Then test the install script. You're done! 🎉

