#!/bin/bash

# 设置颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查是否安装了Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}[错误] 未检测到Go环境，请先安装Go并添加到PATH${NC}"
    exit 1
fi

# 创建输出目录
mkdir -p bin/windows
mkdir -p bin/linux
mkdir -p bin/darwin

# 显示菜单
show_menu() {
    clear
    echo -e "${BLUE}==============================${NC}"
    echo -e "${BLUE}   Go 跨平台编译构建工具${NC}"
    echo -e "${BLUE}==============================${NC}"
    echo ""
    echo -e "${YELLOW}1.${NC} 编译 Windows 版本 (cli.exe)"
    echo -e "${YELLOW}2.${NC} 编译 Linux 版本 (cli)"
    echo -e "${YELLOW}3.${NC} 编译 macOS 版本 (cli)"
    echo -e "${YELLOW}4.${NC} 清理编译文件"
    echo -e "${YELLOW}5.${NC} 退出"
    echo ""
}

# 编译Windows版本
build_windows() {
    echo ""
    echo -e "${BLUE}正在编译Windows版本...${NC}"
    echo ""
    
    export GOOS=windows
    export GOARCH=amd64
    
    go build -o bin/windows/cli.exe \
        config.go \
        connection_mode.go \
        client_framework.go \
        encrypt.go \
        is_admin_windows.go \
        main.go \
        tun_windows.go \
        tun_common.go \
        web_controller.go \
        net_device.go
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}[成功] Windows版本编译完成: bin/windows/cli.exe${NC}"
    else
        echo ""
        echo -e "${RED}[失败] Windows版本编译失败${NC}"
    fi
    
    read -p "按回车键继续..."
}

# 编译Linux版本
build_linux() {
    echo ""
    echo -e "${BLUE}正在编译Linux版本...${NC}"
    echo ""
    
    export GOOS=linux
    export GOARCH=amd64
    
    go build -o bin/linux/cli \
        config.go \
        connection_mode.go \
        client_framework.go \
        encrypt.go \
        is_admin_linux.go \
        main.go \
        tun_linux.go \
        tun_common.go \
        web_controller.go \
        net_device.go
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}[成功] Linux版本编译完成: bin/linux/cli${NC}"
        chmod +x bin/linux/cli
    else
        echo ""
        echo -e "${RED}[失败] Linux版本编译失败${NC}"
    fi
    
    read -p "按回车键继续..."
}

# 编译macOS版本
build_darwin() {
    echo ""
    echo -e "${BLUE}正在编译macOS版本...${NC}"
    echo ""
    
    export GOOS=darwin
    export GOARCH=amd64
    
    go build -o bin/darwin/cli \
        config.go \
        connection_mode.go \
        client_framework.go \
        encrypt.go \
        is_admin_darwin.go \
        main.go \
        tun_darwin.go \
        tun_common.go \
        web_controller.go \
        net_device.go
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}[成功] macOS版本编译完成: bin/darwin/cli${NC}"
        chmod +x bin/darwin/cli
    else
        echo ""
        echo -e "${RED}[失败] macOS版本编译失败${NC}"
    fi
    
    read -p "按回车键继续..."
}

# 清理编译文件
clean_files() {
    echo ""
    echo -e "${YELLOW}正在清理编译文件...${NC}"
    
    rm -f bin/windows/cli.exe
    rm -f bin/linux/cli
    rm -f bin/darwin/cli
    
    echo -e "${GREEN}已清理所有编译文件${NC}"
    read -p "按回车键继续..."
}

# 主循环
while true; do
    show_menu
    read -p "请选择操作 (1-5): " choice
    
    case $choice in
        1)
            build_windows
            ;;
        2)
            build_linux
            ;;
        3)
            build_darwin
            ;;
        4)
            clean_files
            ;;
        5)
            echo -e "${GREEN}退出构建工具${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}无效选择，请重新选择${NC}"
            sleep 2
            ;;
    esac
done
