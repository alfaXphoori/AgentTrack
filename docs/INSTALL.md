# Installation Guide for AgentTrack

AgentTrack is a cross-platform activity tracker built with Go. Choose the installation method that best suits your operating system.

---

## 1. Installation on macOS

### Using Homebrew (Recommended)
You can easily install AgentTrack using Homebrew via our custom tap:
```bash
brew tap yourusername/atrack
brew install atrack
```
*(Replace `yourusername/atrack` with the actual GitHub tap repository once published).*

### Building from Source
If you prefer to build from source, ensure you have [Go installed](https://go.dev/dl/):
```bash
go install github.com/yourusername/atrack@latest
```

---

## 2. Installation on Linux

### Pre-compiled Binaries (Debian/Ubuntu, RPM, etc.)
Download the latest `.deb`, `.rpm`, or `.tar.gz` release from the [GitHub Releases page](#) and install it using your package manager.

**Debian/Ubuntu:**
```bash
wget https://github.com/yourusername/atrack/releases/latest/download/atrack_linux_amd64.deb
sudo dpkg -i atrack_linux_amd64.deb
```

**Arch Linux (AUR):**
*(Coming soon)*
```bash
yay -S atrack
```

### Building from Source
```bash
go install github.com/yourusername/atrack@latest
```

---

## 3. Installation on Windows

### Using Scoop (Recommended)
If you use [Scoop](https://scoop.sh/), you can install AgentTrack easily:
```powershell
scoop bucket add atrack https://github.com/yourusername/atrack-scoop.git
scoop install atrack
```

### Using Winget
*(Coming soon)*
```powershell
winget install atrack
```

### Building from Source
Ensure you have [Go installed](https://go.dev/dl/):
```powershell
go install github.com/yourusername/atrack@latest
```

---

## Verification

After installation, restart your terminal and run:
```bash
atrack version
```
If successful, you will see `AgentTrack version 0.13` (or the latest version).

## Running without Installation (Development)
You can always clone the repository and run the tool directly from the source directory without installing it:
```bash
git clone https://github.com/yourusername/atrack.git
cd atrack
go run . <command>
```
