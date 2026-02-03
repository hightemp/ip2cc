# ip2cc installer for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/hightemp/ip2cc/main/scripts/install.ps1 | iex
# Or: .\install.ps1

$ErrorActionPreference = "Stop"

$Repo = "hightemp/ip2cc"
$BinaryName = "ip2cc"
$InstallDir = "$env:USERPROFILE\bin"

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERROR] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default { 
            # Fallback for older PowerShell
            if ([Environment]::Is64BitOperatingSystem) {
                return "amd64"
            }
            throw "Unsupported architecture: $arch"
        }
    }
}

function Get-LatestVersion {
    $releaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $releaseUrl -UseBasicParsing
        return $release.tag_name
    }
    catch {
        throw "Failed to get latest version: $_"
    }
}

function Get-FileHash256 {
    param([string]$Path)
    $hash = Get-FileHash -Path $Path -Algorithm SHA256
    return $hash.Hash.ToLower()
}

function Verify-Checksum {
    param(
        [string]$FilePath,
        [string]$ExpectedHash
    )
    
    $actualHash = Get-FileHash256 -Path $FilePath
    
    if ($actualHash -ne $ExpectedHash.ToLower()) {
        throw "Checksum verification failed!`nExpected: $ExpectedHash`nActual:   $actualHash"
    }
    
    Write-Info "Checksum verified"
}

function Add-ToPath {
    param([string]$Directory)
    
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$Directory*") {
        $newPath = "$Directory;$currentPath"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$Directory;$env:Path"
        Write-Info "Added $Directory to PATH"
        return $true
    }
    return $false
}

function Main {
    Write-Info "Installing ip2cc for Windows..."
    
    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected architecture: $arch"
    
    # Get latest version
    $version = Get-LatestVersion
    $versionNum = $version.TrimStart('v')
    Write-Info "Latest version: $version"
    
    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Write-Info "Created directory: $InstallDir"
    }
    
    # Archive name format: ip2cc_1.0.0_windows_amd64.zip
    $archiveName = "${BinaryName}_${versionNum}_windows_${arch}.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$version/$archiveName"
    $checksumsUrl = "https://github.com/$Repo/releases/download/$version/checksums.txt"
    
    # Create temp directory
    $tempDir = Join-Path $env:TEMP "ip2cc-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    try {
        $archivePath = Join-Path $tempDir $archiveName
        $checksumsPath = Join-Path $tempDir "checksums.txt"
        
        # Download archive
        Write-Info "Downloading $archiveName..."
        Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        
        # Download checksums
        Write-Info "Downloading checksums..."
        Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing
        
        # Verify checksum
        $checksums = Get-Content $checksumsPath
        $expectedHash = ($checksums | Where-Object { $_ -match $archiveName } | ForEach-Object { ($_ -split '\s+')[0] })
        
        if ($expectedHash) {
            Verify-Checksum -FilePath $archivePath -ExpectedHash $expectedHash
        }
        else {
            Write-Warn "Checksum not found for $archiveName, skipping verification"
        }
        
        # Extract archive
        Write-Info "Extracting..."
        Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force
        
        # Install binary
        $finalPath = Join-Path $InstallDir "$BinaryName.exe"
        Write-Info "Installing to $finalPath..."
        Move-Item -Path (Join-Path $tempDir "$BinaryName.exe") -Destination $finalPath -Force
        
        # Add to PATH
        $pathAdded = Add-ToPath -Directory $InstallDir
        
        Write-Host ""
        Write-Info "Installation successful!"
        
        # Try to run version command
        try {
            & $finalPath version
        }
        catch {
            Write-Warn "Could not run ip2cc. You may need to restart your terminal."
        }
        
        Write-Host ""
        if ($pathAdded) {
            Write-Warn "Please restart your terminal for PATH changes to take effect."
            Write-Host ""
        }
        
        Write-Info "Quick start:"
        Write-Info "  1. Download data: ip2cc update"
        Write-Info "  2. Lookup an IP:  ip2cc 8.8.8.8"
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Main
