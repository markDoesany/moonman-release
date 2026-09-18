param(
    [switch]$PortableOnly
)

$ErrorActionPreference = 'Stop'
$repository = Split-Path -Parent $PSScriptRoot
Set-Location $repository

if ($PortableOnly) {
    wails build -platform windows/amd64 -webview2 browser -o moonman-release.exe
} else {
    if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) {
        throw 'makensis is required for the installer build. Install NSIS and add makensis.exe to PATH, or use -PortableOnly.'
    }
    wails build -platform windows/amd64 -nsis -webview2 browser -o moonman-release.exe
}

$binary = Join-Path $repository 'build\bin\moonman-release.exe'
if (-not (Test-Path -LiteralPath $binary)) {
    throw "Expected build output was not created: $binary"
}

$portableRoot = Join-Path $repository 'build\dist\moonman-release-portable'
$portableZip = Join-Path $repository 'build\dist\MoonmanRelease-windows-amd64-portable.zip'
if (Test-Path -LiteralPath $portableRoot) {
    Remove-Item -LiteralPath $portableRoot -Recurse -Force
}
if (Test-Path -LiteralPath $portableZip) {
    Remove-Item -LiteralPath $portableZip -Force
}

New-Item -ItemType Directory -Path $portableRoot -Force | Out-Null
Copy-Item -LiteralPath $binary -Destination (Join-Path $portableRoot 'moonman-release.exe')
Copy-Item -LiteralPath (Join-Path $repository 'distribution\portable\moonman-release.portable') -Destination $portableRoot
Copy-Item -LiteralPath (Join-Path $repository 'distribution\portable\README.txt') -Destination $portableRoot
New-Item -ItemType Directory -Path (Join-Path $portableRoot 'configs') -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $repository 'distribution\portable\configs\projects.yaml') -Destination (Join-Path $portableRoot 'configs\projects.yaml')
Compress-Archive -Path (Join-Path $portableRoot '*') -DestinationPath $portableZip

Write-Host "Portable package created: $portableZip"
