$ErrorActionPreference = "Stop"

$ScriptRoot = $PSScriptRoot
$Repo = "Shangri-la-0428/oasyce-chain"
$ChainId = if ($env:CHAIN_ID) { $env:CHAIN_ID } else { "oasyce-testnet-1" }
$KeyName = if ($env:KEY_NAME) { $env:KEY_NAME } else { "agent" }
$KeyringBackend = if ($env:KEYRING_BACKEND) { $env:KEYRING_BACKEND } else { "test" }
$HomeDir = if ($env:HOME_DIR) { $env:HOME_DIR } else { Join-Path $HOME ".oasyced" }
$RestUrl = if ($env:REST_URL) { $env:REST_URL } else { "http://47.93.32.88:1317" }
$FaucetUrl = if ($env:FAUCET_URL) { $env:FAUCET_URL } else { "http://47.93.32.88:8080/faucet" }
$RequestFaucet = if ($env:REQUEST_FAUCET) { $env:REQUEST_FAUCET } else { "1" }
$InstallCli = if ($env:INSTALL_CLI) { $env:INSTALL_CLI } else { "1" }

function Write-Info {
    param([string]$Message)
    Write-Host $Message
}

function Get-OasycedPath {
    $cmd = Get-Command oasyced -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    $fallback = Join-Path $HOME ".oasyce\bin\oasyced.exe"
    if (Test-Path $fallback) {
        return $fallback
    }

    return $null
}

function Install-CliIfNeeded {
    $path = Get-OasycedPath
    if ($path) {
        return $path
    }
    if ($InstallCli -ne "1") {
        throw "oasyced not found and INSTALL_CLI=0"
    }

    if ($ScriptRoot) {
        $installer = Join-Path $ScriptRoot "install_oasyced.ps1"
        if (Test-Path $installer) {
            Write-Info "==> oasyced not found; installing CLI first"
            & $installer
            return Get-OasycedPath
        }
    }

    $tmp = Join-Path $env:TEMP "install_oasyced.ps1"
    Invoke-WebRequest -Uri "https://raw.githubusercontent.com/$Repo/main/scripts/install_oasyced.ps1" -OutFile $tmp
    Write-Info "==> oasyced not found; installing CLI first"
    & $tmp
    return Get-OasycedPath
}

function Ensure-Home {
    $clientToml = Join-Path $HomeDir "config\client.toml"
    if (Test-Path $clientToml) {
        return
    }
    New-Item -ItemType Directory -Force -Path $HomeDir | Out-Null
    & $script:oasyced init "oasyce-account-bootstrap" --chain-id $ChainId --home $HomeDir *> $null
}

function Extract-JsonLine {
    param([string[]]$Lines)
    $filtered = @($Lines | Where-Object { $_ -and $_.Trim() })
    [array]::Reverse($filtered)
    foreach ($line in $filtered) {
        $candidate = $line.Trim()
        if ($candidate.StartsWith("{") -and $candidate.EndsWith("}")) {
            try {
                $null = $candidate | ConvertFrom-Json
                return $candidate
            } catch {
            }
        }
    }
    throw "could not extract key JSON"
}

function Ensure-Key {
    $existing = & $script:oasyced keys show $KeyName -a --keyring-backend $KeyringBackend --home $HomeDir 2>$null
    if ($LASTEXITCODE -eq 0 -and $existing) {
        return @{
            address = $existing.Trim()
            mnemonic_file = $null
            created = $false
        }
    }

    Write-Info "==> Creating key: $KeyName"
    $raw = & $script:oasyced keys add $KeyName --keyring-backend $KeyringBackend --home $HomeDir --output json 2>&1
    $jsonLine = Extract-JsonLine -Lines $raw
    $key = $jsonLine | ConvertFrom-Json

    $mnemonicFile = $null
    if ($key.mnemonic) {
        $mnemonicFile = Join-Path $HomeDir "$KeyName.mnemonic"
        Set-Content -Path $mnemonicFile -Value $key.mnemonic -NoNewline
        Write-Info "    Mnemonic saved to: $mnemonicFile"
    }

    return @{
        address = $key.address
        mnemonic_file = $mnemonicFile
        created = $true
    }
}

function Request-Faucet {
    param([string]$Address)
    if ($RequestFaucet -ne "1") {
        return $false
    }
    Write-Info "==> Requesting faucet funds"
    $requestUrl = "$FaucetUrl?address=$Address"
    $null = Invoke-WebRequest -Uri $requestUrl
    return $true
}

$script:oasyced = Install-CliIfNeeded
if (-not $script:oasyced) {
    throw "oasyced still not found after install"
}

Ensure-Home
$keyInfo = Ensure-Key
$funded = Request-Faucet -Address $keyInfo.address

$summary = [ordered]@{
    schema_version = "1"
    chain_id = $ChainId
    key_name = $KeyName
    address = $keyInfo.address
    home = $HomeDir
    keyring_backend = $KeyringBackend
    mnemonic_file = $keyInfo.mnemonic_file
    faucet_requested = $funded
    rest_url = $RestUrl
}

$summary | ConvertTo-Json -Depth 4
