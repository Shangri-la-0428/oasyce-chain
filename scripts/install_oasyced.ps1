$ErrorActionPreference = "Stop"

$Repo = "Shangri-la-0428/oasyce-chain"
$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $HOME ".oasyce\bin" }
$BinName = "oasyced.exe"

function Write-Info {
    param([string]$Message)
    Write-Host $Message
}

function Resolve-Version {
    if ($Version -ne "latest") {
        return $Version
    }

    $api = "https://api.github.com/repos/$Repo/releases/latest"
    $release = Invoke-RestMethod -Uri $api
    if (-not $release.tag_name) {
        throw "Could not resolve latest release tag from GitHub"
    }
    return $release.tag_name
}

function Resolve-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    switch ($arch) {
        "x64" { return "amd64" }
        "arm64" { return "arm64" }
        default { throw "Unsupported Windows architecture: $arch" }
    }
}

function Ensure-Path {
    param([string]$Directory)
    $currentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $segments = @()
    if ($currentUserPath) {
        $segments = $currentUserPath -split ';'
    }
    if ($segments -contains $Directory) {
        return
    }
    $updated = if ($currentUserPath) { "$Directory;$currentUserPath" } else { $Directory }
    [Environment]::SetEnvironmentVariable("Path", $updated, "User")
    if (-not (($env:Path -split ';') -contains $Directory)) {
        $env:Path = "$Directory;$env:Path"
    }
}

$tag = Resolve-Version
$arch = Resolve-Arch
$asset = "oasyced-windows-$arch.exe"
$url = "https://github.com/$Repo/releases/download/$tag/$asset"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$target = Join-Path $InstallDir $BinName

Write-Info "==> Installing oasyced $tag"
Write-Info "    Asset: $asset"
Write-Info "    Target: $target"

Invoke-WebRequest -Uri $url -OutFile $target
Ensure-Path -Directory $InstallDir

$versionText = & $target version 2>$null

Write-Info "==> Installed successfully"
if ($versionText) {
    Write-Info "    Version: $versionText"
}
Write-Info "    PATH updated for current session and future PowerShell sessions"
