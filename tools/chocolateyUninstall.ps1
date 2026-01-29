$ErrorActionPreference = 'Stop';

$package = Get-Package -Name 'Openem-Ingestor' -ErrorAction SilentlyContinue

$serviceName = "OpenEM-Ingestor"

Stop-Service -Name $serviceName

sc.exe delete $serviceName

