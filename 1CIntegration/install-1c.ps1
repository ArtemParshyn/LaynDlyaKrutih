param([string]$BasePath = "$PSScriptRoot\..\1CBase")
$oneC = 'C:\Program Files (x86)\1cv8t\8.3.27.1688\bin\1cv8t.exe'
if (!(Test-Path $oneC)) { throw "1C platform not found: $oneC" }
if (!(Test-Path (Join-Path $BasePath '1Cv8.1CD'))) { throw "1C base not found: $BasePath" }
$epf = Join-Path $PSScriptRoot 'SendStatement.epf'
Start-Process -FilePath $oneC -ArgumentList 'ENTERPRISE','/F',"`"$BasePath`"",'/Execute',"`"$epf`""
Write-Host "Started 1C external processor."
