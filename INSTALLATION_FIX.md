# Installation 404 Error - Fix

## Problem

Getting 404 when running:
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

## Cause

The GitHub repository is **private**. Raw file URLs only work for **public** repositories.

---

## ✅ Solution 1: Make Repository Public (Recommended)

### Steps:

1. Go to: https://github.com/xadnavyaai/vectra-guard
2. Click **"Settings"** (top right)
3. Scroll down to **"Danger Zone"**
4. Click **"Change visibility"**
5. Select **"Make public"**
6. Confirm by typing the repository name

**After this, the install script will work!**

---

## ✅ Solution 2: Use Local Installation (While Private)

Since the repository is private, use local installation instead:

### Option A: Build from Source
```bash
cd /Users/ramachandravikaschamarthi/VectraHub/vectra-guard
go build -o vectra-guard main.go
sudo cp vectra-guard /usr/local/bin/
```

### Option B: Use Pre-built Binary (from dist/)
```bash
cd /Users/ramachandravikaschamarthi/VectraHub/vectra-guard
sudo cp dist/vectra-guard-darwin-arm64 /usr/local/bin/vectra-guard
# Or for Intel Mac:
# sudo cp dist/vectra-guard-darwin-amd64 /usr/local/bin/vectra-guard
```

### Option C: Run Install Script Locally
```bash
cd /Users/ramachandravikaschamarthi/VectraHub/vectra-guard
./install.sh
```

**Note**: The local install.sh script expects binaries on GitHub, so use Option A or B instead.

---

## ✅ Solution 3: Use Go Install (Works Now)

This works even with private repos if you have access:

```bash
# If you're authenticated with GitHub
go install github.com/xadnavyaai/vectra-guard@latest
```

---

## 🎯 Recommended Path Forward

### For Development/Testing (Now):
**Use local installation:**
```bash
cd /Users/ramachandravikaschamarthi/VectraHub/vectra-guard
go build -o vectra-guard main.go
sudo cp vectra-guard /usr/local/bin/vectra-guard
```

### For Public Release (Later):
1. **Make repository public** on GitHub
2. **Upload binaries** to v1.0.0 release
3. **Test install script**:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
   ```

---

## 📋 Quick Commands

### Install Locally Right Now:
```bash
cd /Users/ramachandravikaschamarthi/VectraHub/vectra-guard
go build -o vectra-guard main.go
sudo cp vectra-guard /usr/local/bin/vectra-guard
vectra-guard init
```

### After Making Repo Public:
```bash
# This will work once repo is public
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

---

## 🎉 Result

After making the repo public and uploading binaries, customers can install with:
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

Works anywhere, no dependencies! ✨

