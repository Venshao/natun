@echo off
SETLOCAL EnableDelayedExpansion

:: ������ɫ
color 0A
title UDP Cloud Server ��ƽ̨����ű�

:: ����Ƿ�װ��Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [����] δ��⵽Go�����������Ȱ�װGo������PATH
    pause
    exit /b 1
)

:: �������Ŀ¼
if not exist bin\windows mkdir bin\windows
if not exist bin\linux mkdir bin\linux
if not exist bin\darwin mkdir bin\darwin

:: ���˵�
:menu
cls
echo ==============================
echo    UDP Cloud Server ��ƽ̨����ű�
echo ==============================
echo.
echo 1. ���� Windows �汾 (server.exe)
echo 2. ���� Linux �汾 (server)
echo 3. ���� macOS �汾 (server)
echo 4. ��������ļ�
echo 5. �˳�
echo.
set /p choice="��ѡ����� (1-5): "

if "%choice%"=="1" goto build_win
if "%choice%"=="2" goto build_linux
if "%choice%"=="3" goto build_darwin
if "%choice%"=="4" goto clean
if "%choice%"=="5" exit /b

echo ��Ч���룬������ѡ��
timeout /t 2 >nul
goto menu

:: ========== ����Windows�汾 ==========
:build_win
echo.
echo ���ڱ���Windows�汾...
echo.

set GOOS=windows
set GOARCH=amd64

go build -o bin\windows\server.exe ^
    main.go ^
    relay.go ^
    server_framework.go

if %errorlevel% equ 0 (
    echo.
    echo [�ɹ�] Windows�汾�������: bin\windows\server.exe
) else (
    echo.
    echo [ʧ��] Windows�汾����ʧ��
)

pause
goto menu

:: ========== ����Linux�汾 ==========
:build_linux
echo.
echo ���ڱ���Linux�汾...
echo.

set GOOS=linux
set GOARCH=amd64

go build -o bin\linux\server ^
    main.go ^
    relay.go ^
    server_framework.go

if %errorlevel% equ 0 (
    echo.
    echo [�ɹ�] Linux�汾�������: bin\linux\server
) else (
    echo.
    echo [ʧ��] Linux�汾����ʧ��
)

pause
goto menu

:: ========== ����macOS�汾 ==========
:build_darwin
echo.
echo ���ڱ���macOS�汾...
echo.

set GOOS=darwin
set GOARCH=amd64

go build -o bin\darwin\server ^
    main.go ^
    relay.go ^
    server_framework.go

if %errorlevel% equ 0 (
    echo.
    echo [�ɹ�] macOS�汾�������: bin\darwin\server
) else (
    echo.
    echo [ʧ��] macOS�汾����ʧ��
)

pause
goto menu


:: ========== ��������ļ� ==========
:clean
echo.
echo ������������ļ�...
if exist bin\windows\server.exe del /q bin\windows\server.exe
if exist bin\linux\server del /q bin\linux\server
if exist bin\darwin\server del /q bin\darwin\server
echo ���������б����ļ�
pause
goto menu

ENDLOCAL
