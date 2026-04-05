param(
  [string]$SourceRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\\..")).Path
)

$ErrorActionPreference = "Stop"

$ValidationRoot = Join-Path ([System.IO.Path]::GetTempPath()) "stateguard-install-validation-$PID"
$InstallRoot = Join-Path $ValidationRoot "install"
$ConfigRoot = Join-Path $ValidationRoot "config"

try {
  Write-Host "Running Windows installer validation..."
  & (Join-Path $SourceRoot "install\\windows\\install.ps1") `
    -SourceRoot $SourceRoot `
    -InstallRoot $InstallRoot `
    -ConfigRoot $ConfigRoot `
    -ValidateOnly

  $BinRoot = Join-Path $InstallRoot "bin"
  $ConfigPath = Join-Path $ConfigRoot "safedata.yaml"
  $WrapperPath = Join-Path $BinRoot "stateguard-compose.cmd"

  foreach ($Path in @(
    (Join-Path $BinRoot "stateguardd.exe"),
    (Join-Path $BinRoot "stateguard.exe"),
    (Join-Path $BinRoot "stateguard-dashboard-api.exe"),
    $ConfigPath,
    $WrapperPath
  )) {
    if (-not (Test-Path $Path)) {
      throw "validation failed: missing $Path"
    }
  }

  $Config = Get-Content -Path $ConfigPath -Raw
  foreach ($Needle in @(
    'policy:'
    'validation:'
    'runtime:'
    'project_boundary: labels+compose_project'
  )) {
    if ($Config -notmatch [regex]::Escape($Needle)) {
      throw "validation failed: config missing '$Needle'"
    }
  }

  Write-Host "Windows installer validation passed."
}
finally {
  if (Test-Path $ValidationRoot) {
    Remove-Item -LiteralPath $ValidationRoot -Recurse -Force
  }
}
