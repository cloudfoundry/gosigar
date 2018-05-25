trap {
  write-error $_
  exit 1
}

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Invoke-WebRequest 'https://github.com/stedolan/jq/releases/download/jq-1.5/jq-win64.exe' -OutFile jq.exe
Invoke-WebRequest 'https://golang.org/dl/?mode=json' -OutFile golang.json

$GO_VERSION = $(./jq.exe -r 'map(select(.stable and (.version | split(""".""")[0] == """go1"""))) | .[0].files[] | select(.os == """windows""" and .arch == """amd64""" and .kind == """installer""").version' ./golang.json)

Write-Host "Checking if Go needs to be installed or updated..."
if ((Get-Command 'go.exe' -ErrorAction SilentlyContinue) -ne $null) {
  $version = "$(go.exe version)"
  if ($version -match "go version $GO_VERSION windows\/amd64") {
    Write-Host "Golang $GO_VERSION already installed, skipping download."
    exit 0
  }
}

Write-Host "Installing $GO_VERSION"

Invoke-WebRequest "https://storage.googleapis.com/golang/$GO_VERSION.windows-amd64.msi" ` -UseBasicParsing -OutFile go.msi

$p = Start-Process -FilePath "msiexec" ` -ArgumentList "/passive /norestart /i go.msi" ` -Wait -PassThru

if ($p.ExitCode -ne 0) {
  throw "Golang MSI installation process returned error code: $($p.ExitCode)"
}

Write-Host "Successfully installed go version: $(go version)"
