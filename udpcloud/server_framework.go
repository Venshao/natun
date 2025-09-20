package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/venshao/natun/gjson"
	"github.com/venshao/natun/glog"
)

// HandlerFunc 定义处理函数类型
type HandlerFunc func(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json)

// 路由注册中心
var router = make(map[string]HandlerFunc)

// RegisterHandler 注册路由处理函数
func RegisterHandler(path string, handler HandlerFunc) {
	router[path] = handler
}

// 发送错误响应
func sendError(conn *net.UDPConn, addr *net.UDPAddr, path string, msg string) {
	resp := map[string]interface{}{
		"path":      path,
		"status":    "error",
		"message":   msg,
		"timestamp": time.Now().Unix(),
	}
	sendJSON(conn, addr, resp)
}

// 通用JSON响应（使用gjson序列化）
func sendJSON(conn *net.UDPConn, addr *net.UDPAddr, data interface{}) {
	jsonData := gjson.New(data)
	respBytes, err := jsonData.ToJson()
	if err != nil {
		glog.Errorf("JSON序列化失败: %v", err)
		return
	}

	_, err = conn.WriteToUDP(respBytes, addr)
	if err != nil {
		glog.Errorf("发送响应失败: %v", err)
	}
}

// 启动UDP服务器
func startUDPServer(port int) {
	addr := &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	}
	listen, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(fmt.Sprintf("监听失败: %v", err))
	}
	defer listen.Close()

	glog.Infof("UDP服务已启动，监听地址: %s", addr)

	body := make([]byte, 65536) // 增大缓冲区到64KB，支持更大的UDP包
	for {
		n, clientAddr, err := listen.ReadFromUDP(body)
		if err != nil {
			glog.Errorf("读取数据失败: %v", err)
			continue
		}

		// 复制数据到新的切片，避免goroutine间的数据竞争
		data := make([]byte, n)
		copy(data, body[:n])

		// 异步处理请求
		go func(data []byte, addr *net.UDPAddr) {
			// 检查是否为统一协议的数据包
			if len(data) >= 5 && data[0] == 0x12 && data[1] == 0x34 && data[2] == 0x56 && data[3] == 0x78 {
				mode := data[4]
				if mode == 0x02 {
					// 中转模式: [魔数4B] + [0x02] + [targetId长度(1B)] + [targetId] + [数据长度(2B)] + [数据]
					if len(data) >= 6 {
						targetIdLen := int(data[5])
						if len(data) >= 6+targetIdLen+2 {
							targetId := string(data[6 : 6+targetIdLen])
							dataLen := int(data[6+targetIdLen])<<8 | int(data[6+targetIdLen+1])
							relayData := data[6+targetIdLen+2:]

							// 检查数据长度是否匹配
							if len(relayData) != dataLen {
								glog.Errorf("[RELAY]中转数据长度不匹配: 期望%d, 实际%d", dataLen, len(relayData))
								return
							}

							// 转发数据到目标客户端
							relayDataToClient(listen, targetId, data, addr)
							return
						}
					}
				}
			}

			// 处理JSON协议的控制消息
			content := string(data)

			if !strings.Contains(content, "\"path\":\"relayData\"") && !strings.Contains(content, "\"fragment\":") {
				glog.Debugf("收到来自 %s 的请求: %s", addr.String(), content)
			}
			parseJSON, err := gjson.LoadContent(content)
			if err != nil {
				glog.Warningf("JSON解析失败: %v 原始内容: %s", err, content)
				sendError(listen, addr, "", "无效的JSON格式")
				return
			}

			// 获取请求路径
			path := parseJSON.GetString("path")
			if path == "" {
				sendError(listen, addr, path, "缺少必要字段: path")
				return
			}

			// 路由匹配
			if handler, exists := router[path]; exists {
				handler(listen, addr, path, parseJSON)
			} else {
				sendError(listen, addr, path, "注册中心不支持此请求路径: "+path)
			}
		}(data, clientAddr)
	}
}

// relayDataToClient 转发二进制数据到目标客户端
func relayDataToClient(conn *net.UDPConn, targetId string, data []byte, fromAddr *net.UDPAddr) {
	// 查找目标客户端
	targetClient := clientMap[targetId]
	if targetClient == nil {
		glog.Warningf("[RELAY]目标客户端不存在：%s", targetId)
		return
	}

	// 直接转发原始数据包，targetId已经是正确的目标ID
	_, err := conn.WriteToUDP(data, targetClient.addr)
	if err != nil {
		glog.Errorf("[RELAY]转发数据到客户端 %s 失败：%v", targetId, err)
	} else {
		//glog.Debugf("[RELAY]转发数据%d字节到客户端 %s", len(data), targetId)
	}
}
