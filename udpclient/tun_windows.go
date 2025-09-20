//go:build windows
// +build windows

package main

import (
	"fmt"
	"os/exec"

	"github.com/venshao/natun/glog"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wintun"
)

type WinTunDevice struct {
	wintun  *wintun.Adapter
	session *wintun.Session
	NetDevice
}

func (p *WinTunDevice) Read(b []byte) (n int, err error) {
	packet, err := p.session.ReceivePacket()
	if err != nil {
		return 0, err
	}
	copy(b, packet)
	return len(packet), err
}

func (p *WinTunDevice) Write(b []byte) (n int, err error) {
	// 添加调试信息
	glog.Debugf("[WINTUN]准备写入数据包，长度=%d", len(b))

	// 检查数据包长度
	if len(b) < 20 {
		glog.Warningf("[WINTUN]数据包长度过短: %d", len(b))
		return 0, fmt.Errorf("packet too short")
	}

	// 检查IP版本
	if b[0]>>4 != 4 {
		glog.Warningf("[WINTUN]非IPv4数据包: version=%d", b[0]>>4)
		return 0, fmt.Errorf("not IPv4 packet")
	}

	// 分配缓冲区
	packet, err := p.session.AllocateSendPacket(len(b))
	if err != nil {
		glog.Errorf("[WINTUN]无法分配发送缓冲区: %v", err)
		return 0, err
	}

	// 复制数据到缓冲区
	copy(packet, b)

	// 发送数据包
	p.session.SendPacket(packet)

	glog.Debugf("[WINTUN]成功写入数据包，长度=%d", len(b))
	return len(b), nil
}

func (p *WinTunDevice) Close() error {
	return p.wintun.Close()
}

func CreateTun() NetDevice {
	const (
		adapterName = "NenoTunAdapter" // 网卡名字
	)
	guid := &windows.GUID{
		Data1: uint32(1),
		Data2: uint16(1),
		Data3: uint16(1),
		Data4: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	// 创建或打开设备
	adapter, err := wintun.CreateAdapter(adapterName, "Wintun", guid)
	if err != nil {
		panic(err)
	} else {
		glog.Debugf("[TUN]创建TUN设备成功，%v", adapter)
	}
	// 手动设置 IP
	output, exeErr := exec.Command("netsh", "interface", "ip", "set", "address",
		adapterName, "static", getTunIP(), "mask=255.255.255.0").CombinedOutput()
	if exeErr != nil {
		glog.Errorf("[TUN]设置TUN设备IP失败: %v", exeErr)
		panic(err)
	} else {
		glog.Debugf("[TUN]设置TUN设备IP成功,命令返回:%s", string(output))
	}
	session, err := adapter.StartSession(wintun.RingCapacityMax)
	if err != nil {
		glog.Errorf("[TUN]启动TUN会话失败: %v", err)
		panic(err)
	}

	// 添加路由配置
	routeCmd := exec.Command("route", "add", "10.10.10.0", "mask", "255.255.255.0",
		getTunIP(), "metric", "1")
	_, routeErr := routeCmd.CombinedOutput()
	if routeErr != nil {
		glog.Errorf("[ROUTE]添加路由失败: %v", routeErr)
	} else {
		glog.Debugf("[ROUTE]添加路由成功")
	}

	// 获取MAC地址
	return &WinTunDevice{
		wintun:  adapter,
		session: &session,
	}
}
