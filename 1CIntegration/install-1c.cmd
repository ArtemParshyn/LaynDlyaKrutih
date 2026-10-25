@echo off
setlocal
set "BASE=%~1"
if "%BASE%"=="" set "BASE=%~dp0..\1CBase"
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
if not exist "%~dp0SendStatement.epf" (
  echo SendStatement.epf not found
  exit /b 3
)
start "1C external processor" "%ONEC%" ENTERPRISE /F "%BASE%" /Execute "%~dp0SendStatement.epf"
echo Started SendStatement.epf in %BASE%
endlocal
