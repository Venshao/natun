//go:build darwin
// +build darwin

package main

import (
	"os/exec"

	"github.com/songgao/water"
	"github.com/venshao/natun/glog"
)

type DarwinTunDevice struct {
	ifce *water.Interface
}

func (p *DarwinTunDevice) Read(b []byte) (n int, err error) {
	return p.ifce.Read(b)
}

func (p *DarwinTunDevice) Write(b []byte) (n int, err error) {
	return p.ifce.Write(b)
}

func (p *DarwinTunDevice) Close() error {
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
	output, exeErr := exec.Command("ifconfig", ifce.Name(), getTunIP(), "10.10.10.1", "up").CombinedOutput()
	if exeErr != nil {
		glog.Errorf("[TUN]设置TUN设备IP失败: %v", exeErr)
		panic(err)
	} else {
		glog.Debugf("[TUN]设置TUN设备IP成功,命令返回:%s", string(output))
	}
	// sudo route add -net 10.10.10.0/24 -interface utun<x>
	cmd := exec.Command("route", "-n", "add", "-net", "10.10.10.0/24", "-interface", ifce.Name())
	output, err = cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("[ROUTE] 添加路由失败: %v, 输出: %s", err, string(output))
	} else {
		glog.Debugf("[ROUTE] 添加路由成功: %s", string(output))
	}

	return &DarwinTunDevice{
		ifce: ifce,
	}
}
