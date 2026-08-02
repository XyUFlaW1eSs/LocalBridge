param(
    [string]$Output = "dist\localbridge.exe",
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$outputDir = Split-Path -Parent $Output
if ($outputDir -and -not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
}
go test ./...
go vet ./...
$ldflags = "-s -w"
if ($Version -and $Version -ne "dev") {
    $ldflags = "$ldflags -X github.com/XyUFlaW1eSs/LocalBridge/internal/version.Value=$Version"
}
go build -buildvcs=false -trimpath -ldflags $ldflags -o $Output ./cmd/localbridge
Write-Host "Built $Output"
