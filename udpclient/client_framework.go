package main

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/venshao/natun/gjson"
	"github.com/venshao/natun/glog"
)

type ResponseHandler func(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json)

type NatConnection struct {
	responseHandlerMap map[string]ResponseHandler
	listen             *net.UDPConn
	cancelBeatRoutine  *context.CancelFunc
}

var serverAddr *net.UDPAddr

// initServerAddr 初始化服务器地址
func initServerAddr() {
	cfg := GetConfig()
	serverAddr = &net.UDPAddr{
		IP:   net.ParseIP(cfg.Server.Host),
		Port: cfg.Server.Port,
	}
	glog.Debugf("[SERVER]服务器地址: %s", serverAddr.String())
}

func (p *NatConnection) RegisterResponseHandler(path string, handler ResponseHandler) {
	if p.responseHandlerMap == nil {
		p.responseHandlerMap = make(map[string]ResponseHandler)
	}
	p.responseHandlerMap[path] = handler
}

func (p *NatConnection) StartClient(port int) {
	// 创建UDP监听
	var err error = nil
	p.listen, err = net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	})
	if err != nil {
		panic(err)
	}

	// 启动goroutine处理数据接收
	go func() {
		body := make([]byte, 65536) // 增大缓冲区到64KB，支持更大的UDP包
		magicHeader := []byte{0x12, 0x34, 0x56, 0x78}
		for {
			n, addr, err := p.listen.ReadFromUDP(body)
			if err != nil {
				glog.Warningf("[INNER]读取UDP数据包失败 %v", err)
				break
			}
			if n >= 5 && bytes.Equal(body[:4], magicHeader) {
				// 统一协议头: [魔数4B] + [模式标识(1B)] + [其他数据]
				mode := body[4]
				if mode == 0x01 {
					// 直连模式: [魔数4B] + [0x01] + [数据长度(2B)] + [TUN数据]
					if n >= 7 {
						length := int(body[5])<<8 | int(body[6])
						if n-7 != length {
							glog.Errorf("[TUN]收到的UDP报文长度%d不符合预期%d", n-7, length)
							continue
						}
						//glog.Debugf("[TUN]收到直连 IP 报文 %d 字节，实际 %d 字节", n, length)
						HandleReceivedPacket(p.listen, body[7:n])
					}
				} else if mode == 0x02 {
					// 中转模式: [魔数4B] + [0x02] + [targetId长度(1B)] + [targetId] + [数据长度(2B)] + [数据]
					if n >= 6 {
						targetIdLen := int(body[5])
						if n >= 6+targetIdLen+2 {
							targetId := string(body[6 : 6+targetIdLen])
							dataLen := int(body[6+targetIdLen])<<8 | int(body[6+targetIdLen+1])
							data := body[6+targetIdLen+2 : n]

							// 检查数据长度是否匹配
							if len(data) != dataLen {
								glog.Errorf("[TUN]中转数据长度不匹配: 期望%d, 实际%d", dataLen, len(data))
								continue
							}

							// 检查是否为发给自己的数据
							if targetId == getClientId() {
								// 将数据写入TUN设备
								HandleReceivedPacket(p.listen, data)
								glog.Debugf("[TUN]收到中转数据%d字节", len(data))
							}
						}
					}
				}
			} else {
				// 处理内部通信数据包
				content := string(body[:n])
				//glog.Debugf("[INNER]内部通信数据包,来自 %s 内容: %s", addr.String(), content)
				parseJSON, err := gjson.LoadContent(content)
				if err != nil {
					glog.Warningf("[INNER]JSON解析失败: %v 原始内容: %s", err, content)
					continue
				}

				// 获取请求路径
				path := parseJSON.GetString("path")
				if path == "" {
					glog.Warningf("[INNER]缺少必要字段: path, 原始内容: %s", content)
					continue
				}

				// 路由匹配
				if handler, exists := p.responseHandlerMap[path]; exists {
					handler(p.listen, addr, path, parseJSON)
				} else {
					glog.Warningf("[INNER]不支持的请求路径: %s", path)
				}
			}
		}
		glog.Debug("数据读取协程退出")
	}()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelBeatRoutine = &cancel
	var lastBeatTime int64 = 0
	go func(ctx context.Context) {
		// 循环向注册中心发送保活心跳
		for {
			// 使用 select 监听退出信号
			select {
			case <-ctx.Done():
				glog.Debugf("心跳协程退出")
				return
			default:
				if time.Now().Unix()-lastBeatTime < 3 {
					time.Sleep(time.Millisecond * 1)
					continue
				}
				lastBeatTime = time.Now().Unix()
				// 发送保活心跳
				err := call(p.listen, serverAddr, "ping", map[string]interface{}{
					"id": getClientId(),
				})
				if err != nil {
					glog.Errorf("[INNER]发送ping到注册中心失败 %v", err)
					time.Sleep(time.Second * 1)
					continue
				}
				glog.Debug("[INNER]发送ping到注册中心成功")
			}
		}
	}(ctx)
}

func (p *NatConnection) changePort(port int) {
	if p.cancelBeatRoutine != nil {
		(*p.cancelBeatRoutine)()
		p.cancelBeatRoutine = nil
	}
	err := p.listen.Close()
	if err != nil {
		glog.Errorf("[INNER]关闭UDP连接失败 %v", err)
	}
	glog.Debug("[INNER]已关闭原UDP连接")
	p.StartClient(port)
}
