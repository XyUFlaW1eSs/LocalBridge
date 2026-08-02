param(
    [string]$Executable = "",
    [string]$Config = ""
)

$ErrorActionPreference = "Stop"
$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not $Executable) {
    $packageExecutable = Join-Path $scriptRoot "..\localbridge.exe"
    $repositoryExecutable = Join-Path $scriptRoot "..\dist\localbridge-v0.1.0\localbridge.exe"
    if (Test-Path -LiteralPath $packageExecutable) {
        $Executable = $packageExecutable
    } else {
        $Executable = $repositoryExecutable
    }
}
if (-not (Test-Path -LiteralPath $Executable)) {
    throw "Executable not found: $Executable"
}

if (-not $Config) {
    $packageConfig = Join-Path $scriptRoot "..\config.yaml"
    $packageExample = Join-Path $scriptRoot "..\config.example.yaml"
    $repositoryConfig = Join-Path $scriptRoot "..\configs\config.yaml"
    $repositoryExample = Join-Path $scriptRoot "..\configs\config.example.yaml"
    foreach ($candidate in @($packageConfig, $repositoryConfig, $packageExample, $repositoryExample)) {
        if (Test-Path -LiteralPath $candidate) {
            $Config = $candidate
            break
        }
    }
}
if (-not $Config -or -not (Test-Path -LiteralPath $Config)) {
    $Config = Join-Path $scriptRoot "..\config.example.yaml"
    Write-Warning "Config not found; using example configuration: $Config"
}

Write-Host "Starting LocalBridge in the foreground. Logs are printed below. Press Ctrl+C to stop."
Write-Host "Executable: $Executable"
Write-Host "Config:     $Config"
& $Executable -config $Config
exit $LASTEXITCODE
