# boil Windows Installer
# Save this as install.ps1

$ErrorActionPreference = "Stop"

# Determine architecture
$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ([System.Runtime.InteropServices.Marshal]::SizeOf([System.IntPtr]::Zero) -eq 8) {
        "x86_64"
    } else {
        "arm64"
    }
} else {
    "i386"
}

# Set up variables
$repoOwner = "santiagomed"
$repoName = "boil"
$downloadUrl = "https://github.com/$repoOwner/$repoName/releases/latest/download/boil_Windows_$arch.zip"
$installPath = "$env:LOCALAPPDATA\Programs\boil"

# Create install directory if it doesn't exist
if (-not (Test-Path $installPath)) {
    New-Item -ItemType Directory -Path $installPath | Out-Null
}

try {
    # Download the latest release
    Write-Host "Downloading boil for Windows $arch..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile "$installPath\boil.zip"

    # Extract the ZIP file
    Write-Host "Extracting files..."
    Expand-Archive -Path "$installPath\boil.zip" -DestinationPath $installPath -Force

    # Clean up ZIP file
    Remove-Item "$installPath\boil.zip"

    # Add to PATH if not already present
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$installPath*") {
        [Environment]::SetEnvironmentVariable("Path", $currentPath + ";$installPath", "User")
        Write-Host "Added boil to your PATH."
    }

    Write-Host "boil has been successfully installed to $installPath"
    Write-Host "Please restart your terminal or run 'refreshenv' to use boil."
} catch {
    Write-Host "An error occurred during installation: $_"
    exit 1
}