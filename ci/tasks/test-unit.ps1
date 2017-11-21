$ErrorActionPreference='Stop'
trap {
    write-error $_
    exit 1
}

$env:GOPATH = Join-Path -Path $PWD "gopath"
$env:PATH = $env:GOPATH + "/bin;C:/go/bin;C:/var/vcap/bosh/bin;" + $env:PATH

cd $env:GOPATH/src/github.com/cloudfoundry/gosigar

function NeedsToInstallGo() {
    Write-Host "Checking if Go needs to be installed or updated..."
    if ((Get-Command 'go.exe' -ErrorAction SilentlyContinue) -eq $null) {
        Write-Host "Go.exe not found, Go will be installed"
        return $true
    }
    $version = "$(go.exe version)"
    if ($version -match 'go version go1\.[1-7]\.\d windows\/amd64') {
        Write-Host "Installed version of Go is not supported, Go will be updated"
        return $true
    }
    Write-Host "Found Go version '$version' installed on the system, skipping install"
    return $false
}

if (NeedsToInstallGo) {
    Write-Host "Installing Go 1.8.3"

    Invoke-WebRequest 'https://storage.googleapis.com/golang/go1.8.3.windows-amd64.msi' `
        -UseBasicParsing -OutFile go.msi

    $p = Start-Process -FilePath "msiexec" `
        -ArgumentList "/passive /norestart /i go.msi" `
        -Wait -PassThru
    if ($p.ExitCode -ne 0) {
        throw "Golang MSI installation process returned error code: $($p.ExitCode)"
    }

    Write-Host "Successfully installed go version: $(go version)"
}

go.exe install github.com/cloudfoundry/gosigar/vendor/github.com/onsi/ginkgo/ginkgo
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error installing ginkgo"
    Write-Error $_
    exit 1
}

ginkgo.exe -r -race -keepGoing -skipPackage=psnotify
if ($LASTEXITCODE -ne 0) {
    Write-Host "Gingko returned non-zero exit code: $LASTEXITCODE"
    Write-Error $_
    exit 1
}
