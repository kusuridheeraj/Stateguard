$ErrorActionPreference = "Stop"

$InstallRoot = Join-Path ${env:ProgramFiles} "Stateguard"
$ConfigRoot = Join-Path ${env:ProgramData} "Stateguard"
$ArtifactRoot = Join-Path $ConfigRoot "artifacts"

Write-Host "Preparing Stateguard directories..."
New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigRoot | Out-Null
New-Item -ItemType Directory -Force -Path $ArtifactRoot | Out-Null

Write-Host "Phase 4 installer scaffold complete."
Write-Host "Next phases will place binaries, register services, and install default config."
