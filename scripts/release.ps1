param([string]$Version = "v0.1.0")
$ErrorActionPreference = "Stop"
& .\scripts\build.ps1 -Output "dist\localbridge-$Version.exe"
Write-Host "Release artifact prepared for $Version"
