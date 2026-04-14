# Gastank Windows Installer
# Usage: iwr -useb https://raw.githubusercontent.com/wtfzambo/gastank/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "wtfzambo/gastank"

function Get-LatestTag {
    $url = "https://api.github.com/repos/$Repo/releases/latest"
    $release = Invoke-RestMethod -Uri $url -Headers @{ Accept = "application/vnd.github+json" }
    return $release.tag_name
}

function Install-Gastank {
    $version = Get-LatestTag
    if (-not $version) {
        Write-Error "Could not determine latest release"
        return
    }

    Write-Host "==> Latest release: $version" -ForegroundColor Blue

    $installerName = "gastank-windows-amd64-installer.exe"
    $url = "https://github.com/$Repo/releases/download/$version/$installerName"
    $tmpDir = Join-Path $env:TEMP "gastank-install"
    $installerPath = Join-Path $tmpDir $installerName

    if (-not (Test-Path $tmpDir)) {
        New-Item -ItemType Directory -Path $tmpDir | Out-Null
    }

    Write-Host "==> Downloading $installerName" -ForegroundColor Blue
    Invoke-WebRequest -Uri $url -OutFile $installerPath -UseBasicParsing

    Write-Host "==> Running installer" -ForegroundColor Blue
    Start-Process -FilePath $installerPath -Wait

    # Cleanup
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue

    Write-Host "==> Done" -ForegroundColor Green
}

Install-Gastank
