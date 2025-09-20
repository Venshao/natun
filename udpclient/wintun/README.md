# WinTun 动态链接库说明

## 概述

Linux/MacOS 请忽略本文件夹。本文件夹包含了 Windows 平台下 Neno 客户端所需的 WinTun 动态链接库文件。WinTun 是一个高性能的 Windows TUN/TAP 驱动程序，用于创建虚拟网络接口。

## 文件结构

```
wintun/
├── README.md          # 本说明文件
├── amd64/
│   └── wintun.dll     # AMD64 架构专用版本
├── arm/
│   └── wintun.dll     # ARM 架构专用版本
├── arm64/
│   └── wintun.dll     # ARM64 架构专用版本
└── x86/
    └── wintun.dll     # x86 架构专用版本
```

## 什么是 WinTun？

WinTun 是由 WireGuard 项目开发的高性能 Windows TUN/TAP 驱动程序，具有以下特点：

- **高性能**: 基于 Windows 内核模式驱动，性能优异
- **稳定性**: 经过 WireGuard 项目长期测试，稳定可靠
- **轻量级**: 体积小巧，资源占用少
- **兼容性**: 支持 Windows 7 及以上版本

## 在 Neno 中的作用

Neno 使用 WinTun 来创建虚拟网络接口，实现以下功能：

1. **虚拟局域网**: 创建 10.10.10.x 网段的虚拟网络
2. **数据包转发**: 在虚拟网络和物理网络之间转发数据包
3. **NAT 穿透**: 通过虚拟接口实现 NAT 穿透后的数据通信
4. **网络隔离**: 提供独立的网络命名空间

## 使用方法

### 手动部署

请按以下步骤操作：

#### 步骤 1: 确定系统架构

在命令提示符中运行以下命令查看系统架构：

```cmd
echo %PROCESSOR_ARCHITECTURE%
```

常见架构对应关系：
- `AMD64` → 使用 `amd64/wintun.dll`
- `ARM` → 使用 `arm/wintun.dll`
- `ARM64` → 使用 `arm64/wintun.dll`
- `x86` → 使用 `x86/wintun.dll`

#### 步骤 2: 复制对应文件

将对应架构的 `wintun.dll` 文件复制到程序运行目录：

```cmd
# 示例：复制 AMD64 版本
copy "wintun\amd64\wintun.dll" "程序目录\wintun.dll"
```

#### 步骤 3: 运行程序

确保 `wintun.dll` 与 `cli.exe` 在同一目录下，然后运行程序。

## 系统要求

### 操作系统
- Windows 10 或更高版本（只在Win10和Win11测试过）

### 权限要求
- 管理员权限（用于创建虚拟网络接口）
- 网络配置权限

### 架构支持
- x86 (32位)
- AMD64 (64位)
- ARM (32位)
- ARM64 (64位)

## 技术细节

### 虚拟网络接口

程序会创建一个名为 "NenoTunAdapter" 的虚拟网络接口，配置如下：
- **接口名称**: NenoTunAdapter
- **IP 地址**: 10.10.10.x（x 为 2-252 的随机数）
- **子网掩码**: 255.255.255.0
- **路由**: 自动添加 10.10.10.0/24 网段路由

### 数据包处理

WinTun 驱动程序负责：
1. 接收来自P2P隧道的数据包
2. 将数据包传递给用户态程序
3. 接收用户态程序的数据包
4. 将数据包发送到P2P隧道

## 相关链接

- [WireGuard 项目](https://www.wireguard.com/)

## 支持

如果遇到问题，请：
1. 查看程序日志输出
2. 检查系统事件日志
3. 确认系统环境符合要求