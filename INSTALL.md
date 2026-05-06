# Installation Guide for TrackCLI

TrackCLI is a cross-platform activity tracker built with Go. Follow the instructions below for your specific operating system.

## Prerequisites

Regardless of your OS, you must have **Go (Golang)** installed on your system.
- Download it from the [official Go website](https://go.dev/dl/).
- Verify installation by running: `go version`

---

## 1. Installation on macOS

### Using Homebrew (Recommended)
If you have Homebrew installed, you can install Go and then TrackCLI:
```bash
brew install go
```

### Building from Source
1. Clone or download this repository.
2. Navigate to the project root:
   ```bash
   cd Track_CLI
   ```
3. Install globally:
   ```bash
   go install .
   ```
4. Ensure `~/go/bin` is in your PATH. Add this to your `~/.zshrc` or `~/.bash_profile`:
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

---

## 2. Installation on Linux

### Using Package Manager
- **Ubuntu/Debian:** `sudo apt install golang-go`
- **Fedora:** `sudo dnf install golang`
- **Arch:** `sudo pacman -S go`

### Building from Source
1. Navigate to the project root.
2. Build and install:
   ```bash
   go install .
   ```
3. Add the Go bin directory to your PATH (usually in `~/.bashrc` or `~/.profile`):
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

---

## 3. Installation on Windows

### Using the Installer
Download and run the `.msi` installer from [go.dev](https://go.dev/dl/).

### Building from Source
1. Open **PowerShell** or **Command Prompt**.
2. Navigate to the project root.
3. Install globally:
   ```powershell
   go install .
   ```
4. **Update Environment Variables:**
   - Go usually adds `%USERPROFILE%\go\bin` to your PATH automatically.
   - If not, search for "Edit the system environment variables" in the Start menu.
   - Under "User variables", find `Path` and add `%USERPROFILE%\go\bin`.

---

## Verification

After installation, restart your terminal and run:
```bash
trackcli version
```
If successful, you will see `TrackCLI version 0.12`.

## Running without Installation
You can always run the tool directly from the source directory without installing it:
```bash
go run . <command>
```
