$taskName = "OpenEM-Ingestor"
$binaryPath = "C:\Users\ScopeM_admin\Documents\src\Ingestor\cmd\openem-ingestor-service\openem-ingestor-service.exe"
$action = New-ScheduledTaskAction -Execute $binaryPath -WorkingDirectory "C:\Users\ScopeM_admin\Documents\src\Ingestor\cmd\openem-ingestor-service"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries

$principal = New-ScheduledTaskPrincipal `
    -UserId "$env:USERDOMAIN\$env:USERNAME" `
    -LogonType Interactive `
    -RunLevel Highest
    
$principal = New-ScheduledTaskPrincipal `
    -UserId "$env:USERDOMAIN\$env:USERNAME" `
    -LogonType Interactive `
    -RunLevel Highest
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Force -Principal $principal

Write-Host "Task '$taskName' created in Task Scheduler"