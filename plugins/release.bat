@echo off

set tag=%1
set commitsh=%2

if "%tag%"=="" (
  echo "must specify tag to release"
  exit
)

setlocal enabledelayedexpansion
for /r %%i in (go.mod) do (
  set m=%%~dpi
  set m=!m:%~dp0=!
  set m=!m:\=/!
  if "%commitsh%"=="" (
    hub release create -m "plugins/!m!%tag% release" plugins/!m!%tag%
  ) else (
    hub release create -m "plugins/!m!%tag% release" -t %commitsh% plugins/!m!%tag%
  )
)
endlocal