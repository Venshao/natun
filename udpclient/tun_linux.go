//go:build linux
// +build linux

package main

import (
	"os/exec"

	"github.com/songgao/water"
	"github.com/venshao/natun/glog"
)

type LinuxTunDevice struct {
	ifce *water.Interface
}

func (p *LinuxTunDevice) Read(b []byte) (n int, err error) {
	return p.ifce.Read(b)
}

func (p *LinuxTunDevice) Write(b []byte) (n int, err error) {
	return p.ifce.Write(b)
}

func (p *LinuxTunDevice) Close() error {
	return p.ifce.Close()
}

func CreateTun() NetDevice {
	config := water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{},
	}
	ifce, err := water.New(config)
	if err != nil {
		glog.Fatalf("Failed to create interface: %s", err)
		panic(err)
	}
	glog.Debugf("Interface Name: %s", ifce.Name())
	// 手动设置 IP
	// sudo ip addr add 10.10.10.2/24 dev tun0
	output, exeErr := exec.Command("ip", "addr", "add", getTunIP(), "dev", ifce.Name()).CombinedOutput()
	if exeErr != nil {
		glog.Fatalf("[TUN]设置TUN设备IP失败: %v", exeErr)
		panic(exeErr)
	} else {
		glog.Debugf("[TUN]设置TUN设备IP成功: %s", string(output))
	}
	// sudo ip link set dev tun0 up
	output, exeErr = exec.Command("ip", "link", "set", "dev", ifce.Name(), "up").CombinedOutput()
	// sudo ip route add 10.10.10.0/24 dev tun0
	cmd := exec.Command("ip", "route", "add", "10.10.10.0/24", "dev", ifce.Name())
	output, err = cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("[ROUTE] 添加路由失败: %v, 输出: %s", err, string(output))
	} else {
		glog.Debugf("[ROUTE] 添加路由成功: %s", string(output))
	}

	return &LinuxTunDevice{
		ifce: ifce,
	}
}
