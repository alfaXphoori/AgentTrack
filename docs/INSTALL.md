# Installation Guide for AgentTrack

AgentTrack is a cross-platform activity tracker built with Go. Choose the installation method that best suits your operating system.

---

## 1. Installation on macOS

### Building from Source (Recommended)
Ensure you have [Go installed](https://go.dev/dl/):
```bash
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack .
go install .
```

Or install directly via Go:
```bash
go install github.com/alfaXphoori/AgentTrack@latest
```

### Using Homebrew
*(Coming soon)*
```bash
brew tap alfaXphoori/atrack
brew install atrack
```

---

## 2. Installation on Linux

### Building from Source
```bash
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack .
sudo mv atrack /usr/local/bin/
```

Or install directly via Go:
```bash
go install github.com/alfaXphoori/AgentTrack@latest
```

### Pre-compiled Binaries
Download the latest `.tar.gz` release from the [GitHub Releases page](https://github.com/alfaXphoori/AgentTrack/releases) and extract it:
```bash
tar -xzf atrack_linux_amd64.tar.gz
sudo mv atrack /usr/local/bin/
```

**Arch Linux (AUR):**
*(Coming soon)*
```bash
yay -S atrack
```

---

## 3. Installation on Windows

### Building from Source
Ensure you have [Go installed](https://go.dev/dl/):
```powershell
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack.exe .
```
Then move `atrack.exe` to a directory in your `PATH`.

### Using Scoop
*(Coming soon)*
```powershell
scoop bucket add atrack https://github.com/alfaXphoori/atrack-scoop.git
scoop install atrack
```

### Using Winget
*(Coming soon)*
```powershell
winget install alfaXphoori.atrack
```

---

## Verification

After installation, restart your terminal and run:
```bash
atrack version
```
If successful, you will see `AgentTrack version 0.13.3` (or the latest version).

## Running without Installation (Development)
You can always clone the repository and run the tool directly from the source directory without installing it:
```bash
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go run . <command>
```
