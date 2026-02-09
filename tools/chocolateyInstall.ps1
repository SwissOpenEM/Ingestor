$ErrorActionPreference = 'Stop';

$package = Get-Package -Name 'Openem-Ingestor' -ErrorAction SilentlyContinue



if ($package) {
    Write-Host "Package already installed. Uninstalling first."
    choco uninstall openem-ingestor -y
}

$serviceName = "OpenEM-Ingestor"
try {
    $service = Get-Service -Name $serviceName -ErrorAction Stop
    Write-Host "Service '$serviceName' exists."
    Stop-Service -Name $serviceName

    sc.exe delete $serviceName
} catch {}


$packageName = 'openem-ingestor'


$pp = Get-PackageParameters

if (!$pp['Scicat.Host']) { $pp['Scicat.Host'] = 'https://dacat.psi.ch' }
if (!$pp['Frontend.Host']) { $pp['Frontend.Host'] = 'https://discovery.psi.ch' }
if (!$pp['Keycloak.Host']) { $pp['Keycloak.Host'] = 'https://kc.psi.ch' }
if (!$pp['S3.Host']) { $pp['S3.Host'] = 'https://scopem-openem.ethz.ch' }

# Copy the executable

# Define the string to find and the string to replace it with
$parameters = @{}; 
$parameters["SCICAT_HOST"] = $pp['Scicat.Host']
$parameters["FRONTEND_HOST"] = $pp['Frontend.Host']
$parameters["KEYCLOAK_HOST"] = $pp['Keycloak.Host']
$parameters["S3_HOST"] = $pp['S3.Host']

$locationPairs = $pp['CollectionLocations'] -split ';'
for ($index = 0; $index -lt $locationPairs.Length; $index++) {
    $pair = $locationPairs[$index] -split ':'
    Write-Host "Adding collection Location: $($pair[0]): $($pair[1])"
    $parameters["COLLECTION_LOCATION$($index + 1)"] = $pair[0]
    $parameters["COLLECTION_LOCATION$($index + 1)_PATH"] = $pair[1]
}


$extractPath = "$Env:ChocolateyInstall\lib\$packageName"
$binaryPath = "$extractPath\openem-ingestor-service.exe"

$yamlFilePath = "$extractPath\openem-ingestor-config-template.yaml"
$configFilePath = "$extractPath\openem-ingestor-config.yaml"

Write-Host "Writing config file $configFilePath"
$yamlContent = Get-Content -Path $yamlFilePath -Raw
foreach ($key in $parameters.Keys) { 
    $escapedKey = "\$\{$key\}"
   $yamlContent = $yamlContent -replace $escapedKey, $parameters[$key]
}; 

# # Save the updated content back to the YAML file
Set-Content -Path $configFilePath -Value $yamlContent

Write-Host "Installing $packageName as a service."
# Prompt for the password
$password = Read-Host -Prompt "Enter admin password:" -AsSecureString

# Convert the password to a credential object using the NETWORK SERVICE account
$credential = New-Object System.Management.Automation.PSCredential("NT AUTHORITY\NETWORK SERVICE", $password)

$shawlPath = (get-command shawl).path
$shawlBinPath = "`"$shawlPath`" run --name `"$serviceName`" --cwd `"$extractPath`" -- `"$binarypath`""


New-Service -Name $serviceName -DisplayName $serviceName -BinaryPathName $shawlBinPath -StartupType Automatic -Credential $credential
$target = (get-item (Get-Item $shawlPath).Target).Directory.FullName

Write-Host "Add firewall settings for $($binaryPath)"
icacls $target /grant "NT AUTHORITY\NETWORK SERVICE:(OI)(CI)F" /T
icacls $binaryPath /grant "NT AUTHORITY\NETWORK SERVICE:(OI)(CI)F" /T

sc.exe config $serviceName obj="NT AUTHORITY\Network Service"

Write-Host "Starting Service"
Start-Service $serviceName

Write-Host (Get-Service $serviceName)

Write-Host "openem-ingestor installed successfully!"
