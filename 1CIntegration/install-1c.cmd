@echo off
setlocal
set "BASE=%~1"
if "%BASE%"=="" set "BASE=%~dp0..\1CBase"
set "SITE=%~2"
if "%SITE%"=="" set "SITE=http://127.0.0.1:3000"
set "ONEC=C:\Program Files (x86)\1cv8t\8.3.27.1688\bin\1cv8t.exe"
if not exist "%ONEC%" (
  echo 1C platform not found: %ONEC%
  exit /b 1
)
if not exist "%BASE%\1Cv8.1CD" (
  echo Information base not found: %BASE%
  echo Usage: install-1c.cmd "D:\path\to\1CBase"
  exit /b 2
)
curl.exe -fL "%SITE%/SendStatement.epf" -o "%BASE%\SendStatement.epf"
if errorlevel 1 (
  echo Failed to download SendStatement.epf from %SITE%
  exit /b 3
)
start "1C external processor" "%ONEC%" ENTERPRISE /F "%BASE%" /Execute "%BASE%\SendStatement.epf"
echo Downloaded and started SendStatement.epf in %BASE%
endlocal
