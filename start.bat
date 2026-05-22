@echo off
setlocal enabledelayedexpansion

set ROOT_DIR=%~dp0..
set BBGO_DIR=%ROOT_DIR%
set SAAS_DIR=%ROOT_DIR%\saas
set WEB_DIR=%SAAS_DIR%\web
set MANAGER_DIR=%SAAS_DIR%\manager
set BBGO_BIN=%BBGO_DIR%\build\bbgo\bbgo-slim.exe
set MANAGER_BIN=%SAAS_DIR%\manager\manager.exe

if "%1"=="" goto start
if "%1"=="start" goto start
if "%1"=="stop" goto stop
if "%1"=="build" goto build
if "%1"=="status" goto status
echo Usage: start.bat {start^|stop^|build^|status}
exit /b 1

:start
echo [START] Starting BBGO SaaS...

REM Build bbgo-slim if not exists
if not exist "%BBGO_BIN%" (
    echo [START] Building bbgo-slim...
    if not exist "%BBGO_DIR%\build\bbgo" mkdir "%BBGO_DIR%\build\bbgo"
    pushd "%BBGO_DIR%"
    set GOTOOLCHAIN=local
    go build -tags release -o "%BBGO_BIN%" ./cmd/bbgo
    popd
    echo [START] bbgo-slim built.
) else (
    echo [START] bbgo-slim ready.
)

REM Build manager if not exists
if not exist "%MANAGER_BIN%" (
    echo [START] Building BBGO Manager...
    pushd "%MANAGER_DIR%"
    set GOPROXY=https://goproxy.cn,direct
    set GOTOOLCHAIN=local
    go build -o "%MANAGER_BIN%" .
    popd
    echo [START] Manager built.
) else (
    echo [START] Manager ready.
)

REM Install web deps if needed
if not exist "%WEB_DIR%\node_modules" (
    echo [START] Installing web dependencies...
    pushd "%WEB_DIR%"
    call npm install
    popd
)

REM Start manager
echo [START] Starting Manager on :8090...
pushd "%MANAGER_DIR%"
start /B "BBGO-Manager" set BBGO_BINARY=%BBGO_BIN%&& %MANAGER_BIN%
popd

REM Start web dev
echo [START] Starting Next.js dev server on :3142...
pushd "%WEB_DIR%"
start /B "BBGO-Web" npx next dev --port 3142
popd

echo.
echo [START] All services running:
echo   Manager API:  http://localhost:8090
echo   Web Frontend: http://localhost:3142
echo.
echo Press Ctrl+C to stop...
pause
goto end

:stop
echo [STOP] Stopping all services...
taskkill /FI "WINDOWTITLE eq BBGO-Manager*" /F 2>/dev/null
taskkill /FI "WINDOWTITLE eq BBGO-Web*" /F 2>/dev/null
echo [STOP] Done.
goto end

:build
echo [BUILD] Rebuilding all...
if exist "%BBGO_BIN%" del "%BBGO_BIN%"
if exist "%MANAGER_BIN%" del "%MANAGER_BIN%"
if not exist "%BBGO_DIR%\build\bbgo" mkdir "%BBGO_DIR%\build\bbgo"
pushd "%BBGO_DIR%"
set GOTOOLCHAIN=local
go build -tags release -o "%BBGO_BIN%" ./cmd/bbgo
popd
pushd "%MANAGER_DIR%"
set GOPROXY=https://goproxy.cn,direct
set GOTOOLCHAIN=local
go build -o "%MANAGER_BIN%" .
popd
echo [BUILD] Complete.
goto end

:status
echo [STATUS] Checking services...
tasklist /FI "IMAGENAME eq manager.exe" 2>/dev/null | find /i "manager.exe" >/dev/null && echo Manager: RUNNING || echo Manager: STOPPED
tasklist /FI "IMAGENAME eq node.exe" 2>/dev/null | find /i "node.exe" >/dev/null && echo Web: RUNNING || echo Web: STOPPED
goto end

:end
endlocal
