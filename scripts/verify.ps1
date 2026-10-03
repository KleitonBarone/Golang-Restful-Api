[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
Push-Location (Split-Path $PSScriptRoot -Parent)
$generatedRoot = Join-Path ([IO.Path]::GetTempPath()) ("gardener-swagger-" + [guid]::NewGuid().ToString('N'))
try {
    $env:CGO_ENABLED = '1'
    if ($IsWindows -and -not (Get-Command gcc -ErrorAction SilentlyContinue) -and -not $env:CC) {
        $compiler = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.codex\tools\mingw64\bin\gcc.exe'
        if (-not (Test-Path -LiteralPath $compiler -PathType Leaf)) {
            throw 'Race verification requires a compatible MinGW-w64 compiler. Install it before editing.'
        }
        $env:CC = $compiler
        $env:Path = "$(Split-Path $compiler -Parent)$([IO.Path]::PathSeparator)$env:Path"
    }
    foreach ($arguments in @(@('test', '-race', './...'), @('vet', './...'), @('build', './...'))) {
        & go @arguments
        if ($LASTEXITCODE -ne 0) { throw "go $($arguments -join ' ') failed with exit code $LASTEXITCODE" }
    }
    & go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g router.go -o $generatedRoot --packageName docs
    if ($LASTEXITCODE -ne 0) { throw 'Swagger generation failed' }
    foreach ($name in @('docs.go', 'swagger.json', 'swagger.yaml')) {
        $expected = [IO.File]::ReadAllText((Join-Path $generatedRoot $name)).Replace("`r`n", "`n")
        $actual = [IO.File]::ReadAllText((Join-Path $PWD "docs/$name")).Replace("`r`n", "`n")
        if ($actual -cne $expected) { throw "Generated API documentation is stale: docs/$name" }
    }
    & git diff --check
    if ($LASTEXITCODE -ne 0) { throw 'Git whitespace verification failed' }
}
finally {
    Pop-Location
    $resolvedTemporaryPath = [IO.Path]::GetFullPath($generatedRoot)
    $temporaryRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if (-not $resolvedTemporaryPath.StartsWith($temporaryRoot, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Refusing to remove a path outside the temporary directory'
    }
    if (Test-Path -LiteralPath $resolvedTemporaryPath) { Remove-Item -LiteralPath $resolvedTemporaryPath -Recurse }
}
