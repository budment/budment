$ErrorActionPreference = 'Stop'

function Write-Info($msg) { Write-Host "info: " -ForegroundColor Cyan -NoNewline; Write-Host $msg }
function Write-Success($msg) { Write-Host "ok: " -ForegroundColor Green -NoNewline; Write-Host $msg }
function Write-Warn($msg) { Write-Host "warn: " -ForegroundColor Yellow -NoNewline; Write-Host $msg }
function Write-Err($msg) { Write-Host "error: " -ForegroundColor Red -NoNewline; Write-Host $msg; exit 1 }

Write-Host "`nInstalling Budment CLI for Windows..." -ForegroundColor White

# 1. Architecture detection
$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -match "ARM") { "arm64" } else { "amd64" }
}
else {
    Write-Err "32-bit Windows systems are not supported."
}

# 2. Release metadata retrieval
$Owner = "budment"
$Repo = "budment"

try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Owner/$Repo/releases/latest" -UseBasicParsing
    $LatestTag = $Release.tag_name
}
catch {
    Write-Err "Failed to query latest release from GitHub API: $_"
}

if (-not $LatestTag) {
    Write-Err "Could not resolve valid release tag."
}

$Version = $LatestTag.TrimStart('v')
$FileName = "budment_${Version}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Owner/$Repo/releases/download/$LatestTag/$FileName"

# 3. Create destination workspace
$InstallDir = Join-Path $HOME ".budment\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 4. Download and extract binary
$TempZip = Join-Path $env:TEMP $FileName
Write-Info "Downloading $LatestTag ($FileName)..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing

Expand-Archive -Path $TempZip -DestinationPath $env:TEMP -Force
$ExtractedExe = Join-Path $env:TEMP "budment.exe"
if (-not (Test-Path $ExtractedExe)) {
    Write-Err "Extracted zip does not contain 'budment.exe'."
}

Move-Item -Path $ExtractedExe -Destination (Join-Path $InstallDir "budment.exe") -Force
Remove-Item $TempZip -Force

# 5. Persist PATH in User Environment Variable
$UserPath = [System.Environment]::GetEnvironmentVariable("Path", [System.EnvironmentVariableTarget]::User)
if ($UserPath -notlike "*$InstallDir*") {
    $NewUserPath = if ($UserPath.EndsWith(";")) { "$UserPath$InstallDir" } else { "$UserPath;$InstallDir" }
    [System.Environment]::SetEnvironmentVariable("Path", $NewUserPath, [System.EnvironmentVariableTarget]::User)
    Write-Success "Added $InstallDir to persistent User PATH."
}

# 6. Apply PATH immediately to the active PowerShell session
if ($env:Path -notlike "*$InstallDir*") {
    $env:Path = "$InstallDir;$env:Path"
}

# 7. Completion Banner
Write-Success "Budment CLI $LatestTag installed successfully to $InstallDir\budment.exe"
Write-Host "`nReady to use! Run: budment --help`n" -ForegroundColor Cyan