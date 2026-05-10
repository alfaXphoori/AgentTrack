param()

$ErrorActionPreference = 'Stop'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoDir = Split-Path -Parent $scriptDir

Write-Host '🚀 Installing AgentTrack...'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error 'Go is not installed. Please install Go first.'
    exit 1
}

Push-Location $repoDir
try {
    Write-Host '📦 Building AgentTrack for Windows...'
    go build -o atrack.exe ./cmd/atrack

    Write-Host '🔧 Enabling AgentTrack auto-run...'
    .\atrack.exe autostart install

    Write-Host '▶ Starting the auto-run service for the current session...'
    Start-Process -FilePath (Join-Path $repoDir 'atrack.exe') -ArgumentList 'autostart run' -WindowStyle Hidden -WorkingDirectory $repoDir

    Write-Host ''
    Write-Host '🎉 AgentTrack Installation Complete!'
    Write-Host '--------------------------------------------------------'
    Write-Host 'AgentTrack auto-run has been enabled for Windows.'
} finally {
    Pop-Location
}