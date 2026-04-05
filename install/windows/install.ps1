param(
  [string]$SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\\..")).Path,
  [string]$InstallRoot = (Join-Path ${env:ProgramFiles} "Stateguard"),
  [string]$ConfigRoot = (Join-Path ${env:ProgramData} "Stateguard"),
  [switch]$Force,
  [switch]$ValidateOnly
)

$ErrorActionPreference = "Stop"

if ($ValidateOnly) {
  $ValidationRoot = Join-Path ([System.IO.Path]::GetTempPath()) "stateguard-validate-$PID"
  if (-not $PSBoundParameters.ContainsKey("InstallRoot")) {
    $InstallRoot = Join-Path $ValidationRoot "install"
  }
  if (-not $PSBoundParameters.ContainsKey("ConfigRoot")) {
    $ConfigRoot = Join-Path $ValidationRoot "config"
  }
}

$ArtifactRoot = Join-Path $ConfigRoot "artifacts"
$BinRoot = Join-Path $InstallRoot "bin"
$DistRoot = Join-Path $SourceRoot "dist\\windows"
$DaemonSource = Join-Path $DistRoot "stateguardd.exe"
$CliSource = Join-Path $DistRoot "stateguard.exe"
$ApiSource = Join-Path $DistRoot "stateguard-dashboard-api.exe"
$DaemonTarget = Join-Path $BinRoot "stateguardd.exe"
$CliTarget = Join-Path $BinRoot "stateguard.exe"
$ApiTarget = Join-Path $BinRoot "stateguard-dashboard-api.exe"
$ConfigPath = Join-Path $ConfigRoot "safedata.yaml"
$ComposeWrapper = Join-Path $BinRoot "stateguard-compose.cmd"

function Assert-SourceBinary([string]$Path) {
  if (-not (Test-Path $Path)) {
    throw "Expected built binary not found: $Path. Build release binaries into dist\\windows first."
  }
}

function Assert-FileExists([string]$Path) {
  if (-not (Test-Path $Path)) {
    throw "Expected installer output not found: $Path"
  }
}

function Assert-FileContains([string]$Path, [string[]]$Needles) {
  $Content = Get-Content -Path $Path -Raw
  foreach ($Needle in $Needles) {
    if ($Content -notmatch [regex]::Escape($Needle)) {
      throw "Validation failed: $Path does not contain expected content '$Needle'"
    }
  }
}

Assert-SourceBinary $DaemonSource
Assert-SourceBinary $CliSource
Assert-SourceBinary $ApiSource

Write-Host "Preparing Stateguard directories..."
New-Item -ItemType Directory -Force -Path $InstallRoot, $BinRoot, $ConfigRoot, $ArtifactRoot | Out-Null

Write-Host "Copying binaries..."
Copy-Item -Force $DaemonSource $DaemonTarget
Copy-Item -Force $CliSource $CliTarget
Copy-Item -Force $ApiSource $ApiTarget

if (-not (Test-Path $ConfigPath) -or $Force) {
  @"
version: "1"
project:
  name: stateguard
  environment: production
policy:
  mode: fail-closed
  validation:
    strategy: hybrid
    integrity_check: true
    allow_degraded: false
    restore_test:
      enabled: true
      cadence: periodic
  retention:
    window: 72h
    max_disk_usage_percent: 20
storage:
  local:
    path: $ArtifactRoot
runtime:
  compose:
    project_boundary: labels+compose_project
    live_execution: false
  kubernetes:
    mode: beta
daemon:
  host: 127.0.0.1
  port: 7010
api:
  host: 127.0.0.1
  port: 7011
"@ | Set-Content -Path $ConfigPath -Encoding UTF8
}

@"
@echo off
"$CliTarget" intercept compose %*
"@ | Set-Content -Path $ComposeWrapper -Encoding ASCII

$daemonAction = New-ScheduledTaskAction -Execute $DaemonTarget
$apiAction = New-ScheduledTaskAction -Execute $ApiTarget
$trigger = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest

if ($ValidateOnly) {
  Assert-FileExists $DaemonTarget
  Assert-FileExists $CliTarget
  Assert-FileExists $ApiTarget
  Assert-FileExists $ConfigPath
  Assert-FileExists $ComposeWrapper
  Assert-FileContains $ConfigPath @(
    'policy:'
    'validation:'
    'runtime:'
    'project_boundary: labels+compose_project'
  )
  Write-Host "Validation only mode: installer outputs verified without registering startup tasks."
  Write-Host "Validation root: $ValidationRoot"
  return
}

Write-Host "Registering startup tasks..."
Register-ScheduledTask -TaskName "StateguardDaemon" -Action $daemonAction -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
Register-ScheduledTask -TaskName "StateguardDashboardAPI" -Action $apiAction -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null

Write-Host "Stateguard installed."
Write-Host "CLI: $CliTarget"
Write-Host "Daemon task: StateguardDaemon"
Write-Host "Dashboard task: StateguardDashboardAPI"
