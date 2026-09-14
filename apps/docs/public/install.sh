$ErrorActionPreference = 'Stop'

Write-Host "==> Installing Budment CLI for Windows..." -ForegroundColor Cyan

# Detect CPU architecture
$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -match "ARM") { "arm64" } else { "amd64" }
} else {
    Write-Error "Unsupported architecture: 32-bit is not supported."
    exit 1
}

# Fetch latest version from GitHub releases
$Owner = "budment"
$Repo = "budment"

try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Owner/$Repo/releases" -UseBasicParsing
    $LatestTag = $Release[0].tag_name
} catch {
    Write-Error "Failed to fetch latest release from GitHub."
    exit 1
}

$VersionNoV = $LatestTag.TrimStart('v')
$FileName = "budment_${VersionNoV}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Owner/$Repo/releases/download/$LatestTag/$FileName"

# Create target directory at ~/.budment/bin
$InstallDir = Join-Path $HOME ".budment\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download archive and extract binary
$TempZip = Join-Path $env:TEMP $FileName
Write-Host "Downloading version $LatestTag..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing

Expand-Archive -Path $TempZip -DestinationPath $env:TEMP -Force
Move-Item -Path (Join-Path $env:TEMP "budment.exe") -Destination (Join-Path $InstallDir "budment.exe") -Force
Remove-Item $TempZip -Force

# Update User PATH environment variable if missing
$UserPath = [System.Environment]::GetEnvironmentVariable("Path", [System.EnvironmentVariableTarget]::User)
if ($UserPath -notlike "*$InstallDir*") {
    [System.Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", [System.EnvironmentVariableTarget]::User)
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "`n✔ Added $InstallDir to User PATH." -ForegroundColor Green
}

Write-Host "`n✔ Budment installed successfully!" -ForegroundColor Green
Write-Host "Restart your terminal and run 'budment --help' to get started.`n" -ForegroundColor Yellow