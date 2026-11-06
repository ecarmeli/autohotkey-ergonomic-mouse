<#
.SYNOPSIS
    Validates and builds a complete ErgonomicMouse Windows release locally.

.DESCRIPTION
    Reads the required Go version from go.mod and requires the locally installed
    Go toolchain to match it exactly. It then updates and verifies module metadata,
    checks formatting, runs tests and security/static analysis, builds both Windows
    binaries, verifies their embedded Go build metadata, and compiles the Inno Setup
    installer.

    The script does not run git add, git commit, git push, create a tag, or publish
    a release. Review and commit any go.mod or go.sum changes manually.

.PARAMETER Version
    Application release version to embed in the binaries and installer. Accepts
    either 1.2.3 or v1.2.3. This is the application version, not the Go version.
    If omitted in an interactive session, the script detects the latest published
    GitHub release with the GitHub CLI, suggests the next patch version, and asks
    for confirmation. In a non-interactive session, Version must be supplied.

.PARAMETER OutputDirectory
    Directory for generated binaries. Defaults to bin, matching the release workflow.

.EXAMPLE
    .\scripts\Test-And-Build.ps1 -Version 1.2.3

    Runs all checks and creates the binaries and installer for application version 1.2.3.

.EXAMPLE
    .\scripts\Test-And-Build.ps1

    Detects the latest published GitHub release, suggests the next patch version,
    and requires confirmation before validation and build begin.

.EXAMPLE
    Get-Help .\scripts\Test-And-Build.ps1 -Full

    Displays the complete help for this script.

.NOTES
    Requirements:
      - Windows PowerShell 5.1 or PowerShell 7+
      - Git
      - GitHub CLI (gh), authenticated, when Version is omitted
      - The exact Go version declared by go.mod
      - Repository-managed staticcheck and govulncheck tools
      - Inno Setup 6 at the standard Program Files (x86) path
      - Installer assets expected by installer.iss
#>
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidatePattern('^v?\d+\.\d+\.\d+$')]
    [string]$Version,

    [string]$OutputDirectory = 'bin'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$env:GOTOOLCHAIN = 'local'

function Write-Step {
    param([string]$Message)
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
    }
}

function Install-PinnedAutoHotkey {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepositoryRoot,

        [Parameter(Mandatory = $true)]
        [string]$OutputDirectory
    )

    $workflowPath = Join-Path `
        $RepositoryRoot `
        '.github\workflows\build-and-release.yml'

    if (-not (Test-Path $workflowPath -PathType Leaf)) {
        throw "Build workflow not found: $workflowPath"
    }

    $workflowLines = @(Get-Content -Path $workflowPath)

    $stepName = 'Download and Verify AutoHotkey Core'
    $stepStartIndex = -1
    $stepEndIndex = $workflowLines.Count
    $stepIndentation = $null

    for ($index = 0; $index -lt $workflowLines.Count; $index++) {
        $line = $workflowLines[$index]

        if ($line -match '^(\s*)-\s+name:\s*["'']?(.+?)["'']?\s*$') {
            $currentIndentation = $Matches[1]
            $currentStepName = $Matches[2]

            if ($currentStepName -eq $stepName) {
                if ($stepStartIndex -ge 0) {
                    throw "Workflow step '$stepName' appears more than once in '$workflowPath'."
                }

                $stepStartIndex = $index
                $stepIndentation = $currentIndentation
            }
            elseif (
                $stepStartIndex -ge 0 -and
                $currentIndentation.Length -eq $stepIndentation.Length
            ) {
                $stepEndIndex = $index
                break
            }
        }
    }

    if ($stepStartIndex -lt 0) {
        throw "Workflow step '$stepName' was not found in '$workflowPath'."
    }

    $stepLineCount = $stepEndIndex - $stepStartIndex

    $stepLines = @(
        $workflowLines |
            Select-Object `
                -Skip $stepStartIndex `
                -First $stepLineCount
    )

    $stepContent = $stepLines -join [Environment]::NewLine

    $runBlockFound = $false

    foreach ($line in $stepLines) {
        if ($line -match '^\s*run:\s*\|[+-]?\s*$') {
            $runBlockFound = $true
            break
        }
    }

    if (-not $runBlockFound) {
        throw "Workflow step '$stepName' does not contain a literal run block."
    }

    $urlPattern = '^\s*\$url\s*=\s*"(?<Url>https://github\.com/AutoHotkey/AutoHotkey/releases/download/v(?<UrlVersion>\d+\.\d+\.\d+)/AutoHotkey_(?<AssetVersion>\d+\.\d+\.\d+)\.zip)"\s*$'

    $hashPattern = '^\s*\$expectedHash\s*=\s*"(?<Hash>[A-Fa-f0-9]+)"\s*$'

    $urlMatches = @()
    $hashMatches = @()

    foreach ($line in $stepLines) {
        $urlCandidate = [regex]::Match($line, $urlPattern)

        if ($urlCandidate.Success) {
            $urlMatches += $urlCandidate
        }

        $hashCandidate = [regex]::Match($line, $hashPattern)

        if ($hashCandidate.Success) {
            $hashMatches += $hashCandidate
        }
    }

    if ($urlMatches.Count -eq 0) {
        throw "Unable to find the AutoHotkey URL in workflow step '$stepName'."
    }

    if ($urlMatches.Count -gt 1) {
        throw "Multiple AutoHotkey URLs were found in workflow step '$stepName'."
    }

    if ($hashMatches.Count -eq 0) {
        throw "Unable to find the AutoHotkey SHA-256 in workflow step '$stepName'."
    }

    if ($hashMatches.Count -gt 1) {
        throw "Multiple AutoHotkey SHA-256 values were found in workflow step '$stepName'."
    }

    $urlMatch = $urlMatches[0]
    $hashMatch = $hashMatches[0]

    $parsedHash = $hashMatch.Groups['Hash'].Value

    if ($parsedHash.Length -ne 64) {
        throw (
            "The AutoHotkey SHAme' " +
            "contains $($parsedHash.Length) characters instead of 64."
        )
    }

    $downloadUrl = $urlMatch.Groups['Url'].Value
    $urlVersion = $urlMatch.Groups['UrlVersion'].Value
    $assetVersion = $urlMatch.Groups['AssetVersion'].Value
    $expectedHash = $hashMatch.Groups['Hash'].Value.ToLowerInvariant()

    if ($urlVersion -ne $assetVersion) {
        throw @"
AutoHotkey version mismatch in the build workflow.
Release URL version: $urlVersion
ZIP asset version:   $assetVersion
"@
    }

    $destination = Join-Path $OutputDirectory 'AutoHotkey'
    $archivePath = Join-Path $OutputDirectory "AutoHotkey_$urlVersion.zip"

    Write-Step "Downloading pinned AutoHotkey $urlVersion"

    if (Test-Path $destination) {
        Remove-Item $destination -Recurse -Force
    }

    New-Item `
        -ItemType Directory `
        -Path $destination `
        -Force |
        Out-Null

    try {
        Invoke-WebRequest `
            -Uri $downloadUrl `
            -OutFile $archivePath `
            -UseBasicParsing
    }
    catch {
        throw "Failed to download AutoHotkey $urlVersion. $($_.Exception.Message)"
    }

    try {
        Write-Step 'Verifying AutoHotkey archive'

        $actualHash = (
            Get-FileHash `
                -Path $archivePath `
                -Algorithm SHA256
        ).Hash.ToLowerInvariant()

        if ($actualHash -ne $expectedHash) {
            throw @"
AutoHotkey archive checksum verification failed.
Version:  $urlVersion
Expected: $expectedHash
Actual:   $actualHash
"@
        }

        Write-Host 'AutoHotkey SHA-256 verified.' -ForegroundColor Green

        Write-Step "Extracting AutoHotkey $urlVersion"

        Expand-Archive `
            -Path $archivePath `
            -DestinationPath $destination `
            -Force

        $runtimePath = Join-Path $destination 'AutoHotkey64.exe'

        if (-not (Test-Path $runtimePath -PathType Leaf)) {
            throw "Expected runtime was not found after extraction: $runtimePath"
        }

        $runtimeVersion = (
            Get-Item $runtimePath
        ).VersionInfo.ProductVersion

        if (
            [string]::IsNullOrWhiteSpace($runtimeVersion) -or
            -not $runtimeVersion.StartsWith($urlVersion)
        ) {
            throw "Extracted AutoHotkey runtime version '$runtimeVersion' does not match pinned version '$urlVersion'."
        }

        Write-Host "AutoHotkey $urlVersion staged successfully:" -ForegroundColor Green
        Write-Host "  $runtimePath"
    }
    finally {
        if (Test-Path $archivePath) {
            Remove-Item $archivePath -Force
        }
    }
}

if ($env:OS -ne 'Windows_NT') {
    throw 'This build script must run on Windows.'
}

$repoRoot = (Invoke-Native git rev-parse --show-toplevel | Select-Object -Last 1).Trim()
Set-Location $repoRoot

if (-not (Test-Path 'go.mod' -PathType Leaf)) {
    throw "go.mod was not found at repository root: $repoRoot"
}

if ([string]::IsNullOrWhiteSpace($Version)) {
    if (-not [Environment]::UserInteractive) {
        throw 'Version is required in a non-interactive session. Pass -Version, for example: -Version 1.0.8.'
    }

    $ghCommand = Get-Command gh -ErrorAction SilentlyContinue
    if (-not $ghCommand) {
        throw 'Version was omitted and GitHub CLI (gh) was not found. Install/authenticate gh, or pass -Version explicitly.'
    }

    Write-Step 'Detecting latest published GitHub release'
    $latestTag = (Invoke-Native gh release view --json tagName --jq .tagName | Select-Object -Last 1).Trim()
    if ($latestTag -notmatch '^v?(\d+)\.(\d+)\.(\d+)$') {
        throw "Latest release tag '$latestTag' is not a supported semantic version such as v1.0.7. Pass -Version explicitly."
    }

    $major = [int]$Matches[1]
    $minor = [int]$Matches[2]
    $patch = [int]$Matches[3]
    $suggestedVersion = "$major.$minor.$($patch + 1)"
    $answer = Read-Host "Latest release is $latestTag. Build patch release v$suggestedVersion? [y/N]"
    if ($answer -notmatch '^(?i:y|yes)$') {
        throw 'Build cancelled. Run again and pass -Version to choose a different release version.'
    }
    $Version = $suggestedVersion
}

$normalizedVersion = $Version.TrimStart('v', 'V')
$goDirective = Select-String -Path 'go.mod' -Pattern '^go\s+(\S+)\s*$' | Select-Object -First 1
if (-not $goDirective) {
    throw 'Unable to read the Go version from go.mod.'
}
$requiredGoVersion = $goDirective.Matches[0].Groups[1].Value

Write-Step 'Checking local toolchain'
$goVersionOutput = (Invoke-Native go version | Select-Object -Last 1).Trim()
if ($goVersionOutput -notmatch '^go version go([^\s]+) windows/') {
    throw "Unexpected Go version output: $goVersionOutput"
}
$installedGoVersion = $Matches[1]
if ($installedGoVersion -ne $requiredGoVersion) {
    throw "Go version mismatch. go.mod requires $requiredGoVersion, but local Go is $installedGoVersion."
}
Write-Host "Using Go $installedGoVersion with GOTOOLCHAIN=local"

Write-Step 'Checking repository state'
Invoke-Native git diff --check
$commit = (Invoke-Native git rev-parse --short HEAD | Select-Object -Last 1).Trim()
Write-Host "Commit: $commit"

if (Test-Path $OutputDirectory) {
    Write-Step "Cleaning $OutputDirectory"
    Remove-Item -Recurse -Force $OutputDirectory
}
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

Write-Step 'Updating and verifying module metadata'
Invoke-Native go mod tidy
Invoke-Native go mod verify
if (git status --short -- go.mod go.sum) {
    Write-Warning 'go mod tidy changed go.mod or go.sum. Review and commit those changes before merging or tagging.'
    git diff -- go.mod go.sum
}

Write-Step 'Checking and applying Go formatting'
$unformatted = @(gofmt -l .)

if ($LASTEXITCODE -ne 0) {
    throw 'gofmt failed while checking source files.'
}

if ($unformatted.Count -gt 0) {
    Write-Host 'Formatting the following files:' -ForegroundColor Yellow
    $unformatted | ForEach-Object {
        Write-Host "  $_" -ForegroundColor Yellow
    }

    $gofmtArguments = @('-w') + $unformatted
    Invoke-Native -FilePath 'gofmt' -Arguments $gofmtArguments

    $stillUnformatted = @(gofmt -l .)
    if ($LASTEXITCODE -ne 0) {
        throw 'gofmt failed while verifying formatted source files.'
    }

    if ($stillUnformatted.Count -gt 0) {
        throw "Files remain unformatted after running gofmt: $($stillUnformatted -join ', ')"
    }

    Write-Host 'Go formatting applied successfully.' -ForegroundColor Green
}
else {
    Write-Host 'All Go files are correctly formatted.' -ForegroundColor Green
}

Write-Step 'Running tests and static checks'
Invoke-Native go test ./...
Invoke-Native go vet ./...
Invoke-Native go tool staticcheck ./...
Invoke-Native go tool govulncheck ./...

Write-Step 'Building Windows binaries'
$buildTime = Get-Date -Format 'yyyy-MM-dd_HH:mm:ss'
$ldflags = "-s -w -H=windowsgui -X main.version=$normalizedVersion -X main.gitCommit=$commit -X main.buildTime=$buildTime"
$launcher = Join-Path $OutputDirectory 'Launcher.exe'
$deployManager = Join-Path $OutputDirectory 'DeployManager.exe'
$commonBuildArguments = @(
    'build'
    '-trimpath'
    '-ldflags'
    $ldflags
)

$launcherBuildArguments = $commonBuildArguments + @(
    '-o'
    $launcher
    '.\cmd\launcher\main.go'
)

$deployManagerBuildArguments = $commonBuildArguments + @(
    '-o'
    $deployManager
    '.\cmd\deploymanager\main.go'
)

Invoke-Native -FilePath 'go' -Arguments $launcherBuildArguments
Invoke-Native -FilePath 'go' -Arguments $deployManagerBuildArguments

Write-Step 'Verifying embedded Go build metadata'
foreach ($binary in @($launcher, $deployManager)) {
    if (-not (Test-Path $binary -PathType Leaf)) {
        throw "Expected binary was not created: $binary"
    }
    $metadata = (& go version -m $binary 2>&1 | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "Unable to inspect build metadata for $binary"
    }
    if ($metadata -notmatch [regex]::Escape("go$requiredGoVersion")) {
        throw "$binary was not built with Go $requiredGoVersion."
    }
    Write-Host "$binary OK" -ForegroundColor Green
}

Install-PinnedAutoHotkey `
    -RepositoryRoot $repoRoot `
    -OutputDirectory $OutputDirectory

Write-Step 'Building Inno Setup installer'
$isccPath = Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe'
if (-not (Test-Path $isccPath -PathType Leaf)) {
    throw "Inno Setup compiler not found at: $isccPath. Install Inno Setup 6 before running the complete build."
}
if (-not (Test-Path 'installer.iss' -PathType Leaf)) {
    throw 'installer.iss was not found in the repository root.'
}
Invoke-Native $isccPath "/DAppVersion=$normalizedVersion" "/DAppNumericVersion=$normalizedVersion.0" '.\installer.iss'

Write-Step 'Build summary'
Get-ChildItem -Path $OutputDirectory -File | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
Write-Host "Checks and build completed successfully for v$normalizedVersion." -ForegroundColor Green
