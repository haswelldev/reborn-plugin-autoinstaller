@echo off
:: Build the NSIS installer on Windows.
:: Requirements: NSIS 3.x — https://nsis.sourceforge.io/Download

echo === Building NSIS installer ===
where makensis >nul 2>&1
if errorlevel 1 (
    echo ERROR: makensis not found in PATH.
    echo Install NSIS from https://nsis.sourceforge.io/Download
    pause
    exit /b 1
)

if not exist "dist\reborn-plugin-autoinstaller.exe" (
    echo ERROR: dist\reborn-plugin-autoinstaller.exe not found.
    echo Build the app first, then run this script.
    pause
    exit /b 1
)

if not exist "dist" mkdir dist
makensis installer.nsi
if errorlevel 1 (
    echo ERROR: NSIS build failed.
    pause
    exit /b 1
)

echo.
echo Done! dist\RebornPluginAutoinstaller-Setup.exe is ready.
pause
