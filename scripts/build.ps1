param(
    [string]$Output = "dist\localbridge.exe"
)

$ErrorActionPreference = "Stop"
$outputDir = Split-Path -Parent $Output
if ($outputDir -and -not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
}
go test ./...
go vet ./...
go build -trimpath -ldflags "-s -w" -o $Output ./cmd/localbridge
Write-Host "Built $Output"
