#Requires -Version 7.0

param (
    [ValidateSet('User', 'System')]
    [string]$Mode = '',
    [switch]$Help
)

$ErrorActionPreference = 'Stop'

$REPO = 'barthjs/swim-timer'
$APP_NAME = 'swim-timer'
$APP_DISPLAY_NAME = 'Swim-Timer'

$Colors = @{
    Green = "`e[32m"
    Red = "`e[31m"
    Yellow = "`e[33m"
    Blue = "`e[34m"
    Reset = "`e[0m"
}

function Write-Info
{
    Write-Host "$( $Colors.Blue )ℹ$( $Colors.Reset ) $args"
}

function Write-Success
{
    Write-Host "$( $Colors.Green )✓$( $Colors.Reset ) $args"
}

function Write-Warn
{
    Write-Host "$( $Colors.Yellow )⚠$( $Colors.Reset ) $args"
}

function Write-ErrorMsg
{
    Write-Host "$( $Colors.Red )✗$( $Colors.Reset ) $args" -ForegroundColor Red
}

function Show-Help
{
    @'
Usage: install.ps1 [OPTIONS]

Options:
  -Mode <User|System>  Set installation mode (User or System)
  -Help                Show this help message

If no mode is specified, you will be prompted to choose.

Modes:
  User    Install for current user only
  System  Install system-wide (requires admin, runs as SYSTEM service)
'@ | Write-Host
}

if ($Help)
{
    Show-Help
    exit 0
}

Write-Host "$($Colors.Green)=== ${APP_NAME} installer ===$($Colors.Reset)`n"

$Architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
$ArchMapping = @{
    'X86' = '386'
    'X64' = 'amd64'
    'Arm64' = 'arm64'
}

if (-not $ArchMapping.ContainsKey([string]$Architecture))
{
    Write-ErrorMsg "Unsupported architecture: $Architecture"
    exit 1
}

$Arch = $ArchMapping[[string]$Architecture]
$BinaryName = "${APP_NAME}_windows_${Arch}.zip"

if ( [string]::IsNullOrEmpty($Mode))
{
    Write-Host "Choose installation mode:"
    Write-Host "  1) User       (install for current user only)"
    Write-Host "  2) System     (install for all users, requires admin, runs as SYSTEM)"
    $Choice = Read-Host "Select mode (1 or 2)"

    switch ($Choice)
    {
        '1' {
            $Mode = 'User'
        }
        '2' {
            $Mode = 'System'
        }
        default {
            Write-ErrorMsg "Invalid choice"
            exit 1
        }
    }
}

Write-Info "Installation mode: $Mode"

if ($Mode -eq 'User')
{
    $ConfigDir = Join-Path $env:APPDATA $APP_NAME
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs' $APP_NAME
    $StartMenuDir = Join-Path $env:APPDATA 'Microsoft' 'Windows' 'Start Menu' 'Programs'
    $ShortcutPath = Join-Path $StartMenuDir "$APP_DISPLAY_NAME.lnk"
}
else
{
    $ConfigDir = Join-Path $env:ProgramData $APP_NAME
    $InstallDir = Join-Path $env:ProgramFiles $APP_NAME
    $StartMenuDir = Join-Path $env:ProgramData 'Microsoft' 'Windows' 'Start Menu' 'Programs'
    $ShortcutPath = Join-Path $StartMenuDir "$APP_DISPLAY_NAME.lnk"
}

if ($Mode -eq 'System')
{
    $Principal = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
    if (-not $Principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))
    {
        Write-ErrorMsg "System installation requires administrator privileges"
        exit 1
    }
}

Write-Info "Downloading latest $APP_NAME binary for Windows $Arch..."

try
{
    $Headers = @{ }
    if ($env:GITHUB_TOKEN)
    {
        $Headers['Authorization'] = "token $env:GITHUB_TOKEN"
    }

    $ReleaseResponse = Invoke-RestMethod `
        -Uri "https://api.github.com/repos/$REPO/releases/latest" `
        -Headers $Headers `
        -ErrorAction Stop
}
catch
{
    Write-ErrorMsg "Failed to fetch release information from GitHub: $_"
    exit 1
}

$Asset = $ReleaseResponse.assets | Where-Object { $_.name -eq $BinaryName }
if (-not $Asset)
{
    Write-ErrorMsg "Failed to find binary for $BinaryName"
    exit 1
}

$ReleaseUrl = $Asset.url
$AppVersion = $ReleaseResponse.tag_name

Write-Info "Downloading from: $ReleaseUrl"

$TmpDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "swim-timer-$([guid]::NewGuid() )") -Force
trap
{
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

try
{
    $BinaryPath = Join-Path $TmpDir $BinaryName
    $DownloadHeaders = $Headers.Clone()
    $DownloadHeaders['Accept'] = 'application/octet-stream'

    Invoke-WebRequest `
        -Uri $ReleaseUrl `
        -Headers $DownloadHeaders `
        -OutFile $BinaryPath `
        -ErrorAction Stop

    $ChecksumAsset = $ReleaseResponse.assets | Where-Object { $_.name -like '*checksums*' }
    if ($ChecksumAsset)
    {
        $ChecksumPath = Join-Path $TmpDir 'checksums.txt'
        $ChecksumsFetched = $false
        try
        {
            Invoke-WebRequest `
                -Uri $ChecksumAsset.url `
                -Headers $DownloadHeaders `
                -OutFile $ChecksumPath `
                -ErrorAction Stop
            $ChecksumsFetched = $true
        }
        catch
        {
            Write-Warn "Failed to download checksums, skipping integrity check: $_"
        }

        if ($ChecksumsFetched)
        {
            $BinaryFileName = [System.IO.Path]::GetFileName($BinaryPath)
            $ExpectedLine = Get-Content $ChecksumPath | Where-Object { ($_ -split '\s+')[1] -eq $BinaryFileName } | Select-Object -First 1

            if (-not $ExpectedLine)
            {
                Write-Warn "Checksum entry not found for $BinaryFileName, skipping integrity check"
            }
            else
            {
                $ExpectedChecksum = ($ExpectedLine -split '\s+')[0].ToLower()
                $BinaryChecksum = (Get-FileHash -Path $BinaryPath -Algorithm SHA256).Hash.ToLower()
                if ($BinaryChecksum -eq $ExpectedChecksum)
                {
                    Write-Success "Binary integrity verified"
                }
                else
                {
                    throw "Binary integrity check failed for $BinaryFileName"
                }
            }
        }
    }
    else
    {
        Write-Warn "Checksums file not available, skipping integrity check"
    }

    Expand-Archive -Path $BinaryPath -DestinationPath $TmpDir -Force
    $ExtractedBinary = Join-Path $TmpDir "$APP_NAME.exe"
    if (-not (Test-Path $ExtractedBinary))
    {
        Write-ErrorMsg "Failed to extract binary from archive"
        exit 1
    }

    Write-Info "Installing binary..."
    if (-not (Test-Path $InstallDir))
    {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    if (-not (Test-Path $ConfigDir))
    {
        New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
    }

    $FinalBinaryPath = Join-Path $InstallDir "$APP_NAME.exe"
    Copy-Item -Path $ExtractedBinary -Destination $FinalBinaryPath -Force
    Write-Success "Binary installed to $FinalBinaryPath"

    if (-not (Test-Path $StartMenuDir))
    {
        New-Item -ItemType Directory -Path $StartMenuDir -Force | Out-Null
    }

    $WshShell = New-Object -ComObject WScript.Shell
    $Shortcut = $WshShell.CreateShortcut($ShortcutPath)
    $Shortcut.TargetPath = $FinalBinaryPath
    $Shortcut.WorkingDirectory = $ConfigDir
    $Shortcut.Description = "$APP_DISPLAY_NAME"
    $Shortcut.Save()
    Write-Success "Shortcut created at $ShortcutPath"

    Write-Host ""
    $ServiceChoice = Read-Host "Do you want to install as a Windows Service? (Y/n)"

    if ($ServiceChoice -match '^[Yy]$|^$')
    {
        Write-Info "Installing Windows Service..."

        $ServiceName = "SwimTimer"
        $ServiceDisplayName = "Swim-Timer"

        $ExistingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($ExistingService)
        {
            Write-Warn "Service already exists, updating..."
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            Start-Sleep -Milliseconds 500
            Remove-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        }

        # The binary has no Windows service handler of its own, so NSSM runs it as a service.
        if (-not (Get-Command nssm -ErrorAction SilentlyContinue))
        {
            Write-Warn "The service requires NSSM, which was not found."
            Write-Host "Install it, for example with: winget install nssm"
            Write-Host "Then run this script again to install the service."
        }
        else
        {
            Write-Info "Using NSSM to install service..."
            nssm install $ServiceName $FinalBinaryPath
            nssm set $ServiceName DisplayName $ServiceDisplayName
            nssm set $ServiceName AppDirectory $ConfigDir
            nssm set $ServiceName AppNoConsole 1
            nssm set $ServiceName AppRestartDelay 5000
            # Tells the application that NSSM restarts it, so it does not start a second instance after an update.
            nssm set $ServiceName AppEnvironmentExtra SWIM_TIMER_SUPERVISED=1

            Write-Success "Service created: $ServiceName"

            Write-Host ""
            $StartServiceChoice = Read-Host "Enable and start the service now? (Y/n)"

            if ($StartServiceChoice -match '^[Yy]$|^$')
            {
                Write-Info "Starting service..."
                Set-Service -Name $ServiceName -StartupType Automatic
                Start-Service -Name $ServiceName
                Write-Success "Service enabled and started"
                Write-Host "Check status with: Get-Service $ServiceName"
            }
        }
    }
    else
    {
        Write-Info "Skipped service installation"
        Write-Host ""
        Write-Host "You can run $APP_NAME by:"
        Write-Host "  1. Clicking the shortcut in Start Menu"
        Write-Host "  2. Running: $FinalBinaryPath"
    }

    Write-Host ""
    Write-Success "Installation complete!"
    Write-Host "Start the app from the Start Menu or by running: $APP_NAME"
}
catch
{
    Write-ErrorMsg "Installation failed: $_"
    exit 1
}
finally
{
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
