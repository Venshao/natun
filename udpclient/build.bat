@echo off
SETLOCAL EnableDelayedExpansion

:: 设置颜色
color 0A
title Go 三平台交叉编译脚本

:: 检查是否安装了Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未检测到Go环境，请先安装Go并配置PATH
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
echo    Go 三平台交叉编译工具
echo ==============================
echo.
echo 1. 编译 Windows 版本 (cli.exe)
echo 2. 编译 Linux 版本 (cli)
echo 3. 编译 macOS 版本 (cli)
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

go build -o bin\windows\cli.exe ^
    config.go ^
	connection_mode.go ^
    client_framework.go ^
    encrypt.go ^
    is_admin_windows.go ^
    main.go ^
    tun_windows.go ^
    tun_common.go ^
    web_controller.go ^
    net_device.go

if %errorlevel% equ 0 (
    echo.
    echo [成功] Windows版本已生成: bin\windows\cli.exe
) else (
    echo.
    echo [失败] Windows版本编译出错
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

go build -o bin\linux\cli ^
    config.go ^
	connection_mode.go ^
    client_framework.go ^
    encrypt.go ^
    is_admin_linux.go ^
    main.go ^
    tun_linux.go ^
    tun_common.go ^
    web_controller.go ^
    net_device.go

if %errorlevel% equ 0 (
    echo.
    echo [成功] Linux版本已生成: bin\linux\cli
) else (
    echo.
    echo [失败] Linux版本编译出错
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

go build -o bin\darwin\cli ^
    config.go ^
	connection_mode.go ^
    client_framework.go ^
    encrypt.go ^
    is_admin_darwin.go ^
    main.go ^
    tun_darwin.go ^
    tun_common.go ^
    web_controller.go ^
    net_device.go

if %errorlevel% equ 0 (
    echo.
    echo [成功] macOS版本已生成: bin\darwin\cli
) else (
    echo.
    echo [失败] macOS版本编译出错
)

pause
goto menu


:: ========== 清理编译文件 ==========
:clean
echo.
echo 正在清理编译文件...
if exist bin\windows\cli.exe del /q bin\windows\cli.exe
if exist bin\linux\cli del /q bin\linux\cli
if exist bin\darwin\cli del /q bin\darwin\cli
echo 已清理所有编译生成文件
pause
goto menu

ENDLOCAL