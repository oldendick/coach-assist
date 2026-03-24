# Coach Assist Windows Installer (PowerShell)
# Usage: powershell -c "irm https://raw.githubusercontent.com/oldendick/coach-assist/main/install.ps1 | iex"

$ErrorActionPreference = "Stop"

$Repo = "oldendick/coach-assist"
$AppName = "coachassist"

Write-Host "--- Coach Assist Installer (Windows) ---" -ForegroundColor Cyan
Write-Host "Installing into: $(Get-Location)"

# 1. Detect Architecture
$Arch = $env:PROCESSOR_ARCHITECTURE.ToLower()
if ($Arch -eq "amd64") {
    $Platform = "windows-amd64"
} else {
    Write-Host "Unsupported Windows architecture: $Arch" -ForegroundColor Red
    exit 1
}

# 2. Fetch Latest Release via GitHub API
Write-Host "[1/4] Fetching latest release info from GitHub..."
try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name
} catch {
    Write-Host "Error: Could not determine latest version from GitHub API." -ForegroundColor Red
    exit 1
}

# Find Asset URL (matching platform and zip)
$Asset = $Release.assets | Where-Object { $_.name -like "*$Platform.zip" } | Select-Object -First 1
if ($null -eq $Asset) {
    Write-Host "Error: Could not find asset for $Platform in release $Version." -ForegroundColor Red
    exit 1
}

$AssetUrl = $Asset.browser_download_url

# 3. Download
$DestFile = "coach-assist-$Version.zip"
Write-Host "[2/4] Downloading $Version ($Platform)..."
Invoke-WebRequest -Uri $AssetUrl -OutFile $DestFile

# 4. Extract
Write-Host "[3/4] Extracting $Version..."
$ExtractDir = "coach-assist-$Version"
Expand-Archive -Path $DestFile -DestinationPath "." -Force

# Cleanup Zip
Remove-Item -Path $DestFile

Write-Host ""
Write-Host "--- Installation Successful! ---" -ForegroundColor Green
Write-Host "Coach Assist $Version has been installed to: $(Get-Location)\$ExtractDir"
Write-Host ""
Write-Host "Next Steps:"
Write-Host "1. Configure the app:"
Write-Host "   - cd $ExtractDir; Copy-Item config.example.json config.json"
Write-Host "   - Edit 'config.json' with your coach details."
Write-Host "2. Run the application:"
Write-Host "   - .\$AppName.exe"
Write-Host "   - (The app will automatically guide you through the Google Workspace client setup on your first run)"
Write-Host ""
