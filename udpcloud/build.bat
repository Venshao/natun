@echo off
SETLOCAL EnableDelayedExpansion

:: 设置颜色
color 0A
title UDP Cloud Server 多平台编译脚本

:: 检查是否安装了Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未检测到Go编译器，请先安装Go并配置PATH
    pause
    exit /b 1
)

:: 创建输出目录
if not exist bin\windows mkdir bin\windows
if not exist bin\linux mkdir bin\linux
if not exist bin\darwin mkdir bin\darwin

:: 主菜单
:menu
cls
echo ==============================
echo    UDP Cloud Server 多平台编译脚本
echo ==============================
echo.
echo 1. 编译 Windows 版本 (server.exe)
echo 2. 编译 Linux 版本 (server)
echo 3. 编译 macOS 版本 (server)
echo 4. 清理编译文件
echo 5. 退出
echo.
set /p choice="请选择操作 (1-5): "

if "%choice%"=="1" goto build_win
if "%choice%"=="2" goto build_linux
if "%choice%"=="3" goto build_darwin
if "%choice%"=="4" goto clean
if "%choice%"=="5" exit /b

echo 无效输入，请重新选择
timeout /t 2 >nul
goto menu

:: ========== 编译Windows版本 ==========
:build_win
echo.
echo 正在编译Windows版本...
echo.

set GOOS=windows
set GOARCH=amd64

go build -o bin\windows\server.exe ^
    main.go ^
    relay.go ^
    server_framework.go

if %errorlevel% equ 0 (
    echo.
    echo [成功] Windows版本编译完成: bin\windows\server.exe
) else (
    echo.
    echo [失败] Windows版本编译失败
)

pause
goto menu

:: ========== 编译Linux版本 ==========
:build_linux
echo.
echo 正在编译Linux版本...
echo.

set GOOS=linux
set GOARCH=amd64

go build -o bin\linux\server ^
    main.go ^
    relay.go ^
    server_framework.go

if %errorlevel% equ 0 (
    echo.
    echo [成功] Linux版本编译完成: bin\linux\server
) else (
    echo.
    echo [失败] Linux版本编译失败
)

pause
goto menu

:: ========== 编译macOS版本 ==========
:build_darwin
echo.
echo 正在编译macOS版本...
echo.

set GOOS=darwin
set GOARCH=amd64

go build -o bin\darwin\server ^
    main.go ^
    relay.go ^
    server_framework.go

if %errorlevel% equ 0 (
    echo.
    echo [成功] macOS版本编译完成: bin\darwin\server
) else (
    echo.
    echo [失败] macOS版本编译失败
)

pause
goto menu


:: ========== 清理编译文件 ==========
:clean
echo.
echo 正在清理编译文件...
if exist bin\windows\server.exe del /q bin\windows\server.exe
if exist bin\linux\server del /q bin\linux\server
if exist bin\darwin\server del /q bin\darwin\server
echo 已清理所有编译文件
pause
goto menu

ENDLOCAL
