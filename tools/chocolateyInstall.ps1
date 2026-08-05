$ErrorActionPreference = 'Stop';

$package = Get-Package -Name 'Openem-Ingestor' -ErrorAction SilentlyContinue

if ($package) {
    Write-Host "Package already installed. Uninstalling first."
    choco uninstall openem-ingestor -y
}

$packageName = 'openem-ingestor'

$pp = Get-PackageParameters


if (!$pp['environment']) { 
    Write-Error -Message "No environment specified (dev, qa, prod)" 
    Exit -1
}

$deployment_env = $pp['Environment']

# Define the string to find and the string to replace it with
$parameters = @{}; 

if ($deployment_env -eq "dev"){
    $parameters["SCICAT_HOST"] = "https://scopem-openem2.ethz.ch/scicat/backend"
    $parameters["FRONTEND_HOST"] = "https://scopem-openem2.ethz.ch"
    $parameters["KEYCLOAK_HOST"] = "https://scopem-openem2.ethz.ch/keycloak"
    $parameters["KEYCLOAK_REALM"] = "facility"
    $parameters["S3_HOST"] = "https://scopem-openem2.ethz.ch"
} elseif ($deployment_env -eq "qa") {
    $parameters["SCICAT_HOST"] = "https://dacat-qa.psi.ch"
    $parameters["FRONTEND_HOST"] = "https://discovery-qa.psi.ch"
    $parameters["KEYCLOAK_HOST"] = "https://kc.psi.ch"
    $parameters["KEYCLOAK_REALM"] = "awi"
    $parameters["S3_HOST"] = "https://scopem-openem.ethz.ch/qa"
} elseif ($deployment_env -eq "prod") {
    $parameters["SCICAT_HOST"] = "https://dacat.psi.ch"
    $parameters["FRONTEND_HOST"] = "https://discovery.psi.ch"
    $parameters["KEYCLOAK_HOST"] = "https://kc.psi.ch"
    $parameters["KEYCLOAK_REALM"] = "awi"
    $parameters["S3_HOST"] = "https://scopem-openem.ethz.ch"
}else{
    Write-Error -Message "Unknown environment specified (allowed: dev, qa, prod)" 
    Exit -1
}

if ($pp['Scicat.Host']) { $parameters['SCICAT_HOST'] = $pp['Scicat.Host'] }
if ($pp['Frontend.Host']) { $parameters['FRONTEND_HOST'] = $pp['Frontend.Host'] }
if ($pp['Keycloak.Host']) { $parameters['KEYCLOAK_HOST'] = $pp['Keycloak.Host'] }
if ($pp['Keycloak.Realm']) { $parameters['KEYCLOAK_REALM'] = $pp['Keycloak.Realm'] }
if ($pp['S3.Host']) { $parameters['S3_HOST'] = $pp['S3.Host'] }

$locationPairs = $pp['CollectionLocations'] -split ';'
for ($index = 0; $index -lt $locationPairs.Length; $index++) {
    $pair = $locationPairs[$index] -split ':'
    Write-Host "Adding collection Location: $($pair[0]): $($pair[1])"
    $parameters["COLLECTION_LOCATION$($index + 1)"] = $pair[0]
    $parameters["COLLECTION_LOCATION$($index + 1)_PATH"] = $pair[1]
}


$extractPath = "$Env:ChocolateyInstall\lib\$packageName"
$binaryPath = "$extractPath\OpenEM-Ingestor.exe"
$iconPath = "$extractPath\openem.ico"

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

Write-Host "Creating shortcut"

Install-ChocolateyShortcut `
  -ShortcutFilePath $env:PUBLIC\Desktop\OpenEM-Ingestor.lnk `
  -TargetPath $binaryPath `
  -WorkingDirectory (Split-Path $binaryPath) `
  -IconPath $iconPath

Write-Host "openem-ingestor installed successfully!"
