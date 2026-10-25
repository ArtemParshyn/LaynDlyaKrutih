param(
  [string]$BasePath = "$PSScriptRoot\..\1CBase",
  [string]$SiteUrl = "http://127.0.0.1:3000"
)
$oneC = 'C:\Program Files (x86)\1cv8t\8.3.27.1688\bin\1cv8t.exe'
if (!(Test-Path $oneC)) { throw "1C platform not found: $oneC" }
if (!(Test-Path (Join-Path $BasePath '1Cv8.1CD'))) { throw "1C base not found: $BasePath" }
$epf = Join-Path $BasePath 'SendStatement.epf'
Invoke-WebRequest "$($SiteUrl.TrimEnd('/'))/SendStatement.epf" -OutFile $epf
Start-Process -FilePath $oneC -ArgumentList 'ENTERPRISE','/F',"`"$BasePath`"",'/Execute',"`"$epf`""
Write-Host "Downloaded and started SendStatement.epf in $BasePath"
