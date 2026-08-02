param([string]$Version = "v0.1.0")
$ErrorActionPreference = "Stop"
$releaseRoot = Join-Path (Get-Location) "dist\localbridge-$Version"
$archive = Join-Path (Get-Location) "dist\localbridge-$Version.zip"
$checksum = "$archive.sha256"

if (Test-Path -LiteralPath $releaseRoot) {
    Remove-Item -LiteralPath $releaseRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $releaseRoot -Force | Out-Null

& .\scripts\build.ps1 -Output (Join-Path $releaseRoot "localbridge.exe") -Version $Version
if (-not (Test-Path -LiteralPath (Join-Path $releaseRoot "localbridge.exe"))) {
    throw "Release executable was not created: $releaseRoot\localbridge.exe"
}
Copy-Item .\configs\config.example.yaml $releaseRoot
Copy-Item .\README.md, .\README.zh-CN.md $releaseRoot
Copy-Item .\LICENSE, .\LICENSE.zh-CN.md $releaseRoot
Copy-Item .\shortcut $releaseRoot -Recurse
Copy-Item .\docs $releaseRoot -Recurse
$packageScripts = Join-Path $releaseRoot "scripts"
New-Item -ItemType Directory -Path $packageScripts -Force | Out-Null
Copy-Item .\scripts\run.ps1 $packageScripts

if (Test-Path -LiteralPath $archive) {
    Remove-Item -LiteralPath $archive -Force
}
Compress-Archive -Path (Join-Path $releaseRoot "*") -DestinationPath $archive -Force
$hash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
"$hash  $(Split-Path -Leaf $archive)" | Set-Content -LiteralPath $checksum -Encoding ascii
Write-Host "Release package prepared: $archive"
Write-Host "SHA-256: $checksum"
