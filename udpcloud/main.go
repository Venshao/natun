package main

import (
	"net"
	"time"

	"github.com/venshao/natun/gjson"
	"github.com/venshao/natun/glog"
)

// NatSession NAT打洞会话
type NatSession struct {
	peerOneId   string
	peerOneAddr *net.UDPAddr
	peerOneConn *net.UDPConn
	peerTwoId   string
	peerTwoAddr *net.UDPAddr
	peerTwoConn *net.UDPConn
}

type Client struct {
	addr         *net.UDPAddr
	conn         *net.UDPConn
	lastBeatTime int64
}

// 客户端id -> 客户端信息的映射关系
var clientMap = make(map[string]*Client)

// 客户端id -> NatSession的映射关系
var natSessionMap = make(map[string]*NatSession)

// 客户端的心跳检测
func pingHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	beatTime := time.Now().Unix()
	resp := map[string]interface{}{
		"path":       "pong",
		"clientIp":   addr.IP.String(),
		"clientPort": addr.Port,
		"timestamp":  beatTime,
	}
	sendJSON(conn, addr, resp)
	id := json.GetString("id")
	client := clientMap[id]
	if client == nil {
		clientMap[id] = &Client{
			addr:         addr,
			conn:         conn,
			lastBeatTime: beatTime,
		}
	} else {
		if client.addr.String() != addr.String() {
			glog.Warningf("收到来自客户端id=%s的心跳, ip=%s, 但是之前已经存在ip=%s，通知其对等节点与之断开", id, addr.String(), client.addr.String())
			// 需要通知当前客户端的对等节点断开与当前客户端的连接
			session := natSessionMap[id]
			if session != nil {
				notifyPeerDisconnect(session, id)
			}
		}
		client.addr = addr
		client.conn = conn
		client.lastBeatTime = beatTime
	}
	glog.Debugf("收到来自客户端id=%s的心跳, ip=%s ", id, addr.String())
}

// 客户端更换端口后会调用此接口
func notifyChangePortHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	srcId := json.GetString("srcId")
	targetId := json.GetString("targetId")
	targetPassword := json.GetString("tp")
	if srcId == "" || targetId == "" || srcId == targetId || targetPassword == "" {
		return
	}
	// 刷新客户端地址信息
	refreshClientInfo(conn, addr, srcId)
	srcClient := clientMap[srcId]
	targetClient := clientMap[targetId]
	if srcClient == nil {
		glog.Warningf("通过 srcId %s 无法找到对应的客户端，忽略", srcId)
		return
	}
	if targetClient == nil {
		glog.Warningf("通过 targetId %s 无法找到对应的客户端，忽略", targetId)
		return
	}
	// 向target节点发送changePort命令
	sendJSON(targetClient.conn, targetClient.addr, map[string]interface{}{
		"path": "changePort",
		"p":    targetPassword,
	})
	glog.Debugf("已向%s发送changePort命令", targetId)
	// 记录当前NAT会话状态
	session := &NatSession{
		peerOneId:   srcId,
		peerOneAddr: srcClient.addr,
		peerOneConn: srcClient.conn,
		peerTwoId:   targetId,
		peerTwoAddr: targetClient.addr,
		peerTwoConn: targetClient.conn,
	}
	natSessionMap[srcId] = session
	natSessionMap[targetId] = session
}

// target 节点收到changePort命令后，会更换端口然后再回调此接口
func portChangedHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	targetId := json.GetString("clientId")
	glog.Debugf("收到 %s 的端口已变更回调", targetId)
	// 刷新target客户端地址信息
	refreshClientInfo(conn, addr, targetId)
	// 走到这里说明被通知更换端口的客户端已经更换完毕端口，开始通知双方打洞
	session := natSessionMap[targetId]
	if session == nil {
		glog.Warningf("通过 targetId %s 无法找到对应NAT会话，忽略", targetId)
		return
	}
	// 也要更新NAT会话信息中被动方的连接信息
	session.peerTwoConn = conn
	session.peerTwoAddr = addr
	// 通知客户端双方同时连接对方
	sendTraverseCommand(session)
}

func sendTraverseCommand(session *NatSession) {
	srcClient := clientMap[session.peerOneId]
	targetClient := clientMap[session.peerTwoId]
	if srcClient == nil || targetClient == nil {
		glog.Warning("srcClient 或 targetClient 为空，不通知打洞")
		return
	}
	glog.Debugf("开始通知 %s 和 %s 双方打洞", session.peerOneId, session.peerTwoId)
	srcClientData := buildNatClientJson(srcClient, session.peerOneId)
	targetClientData := buildNatClientJson(targetClient, session.peerTwoId)
	sendJSON(srcClient.conn, srcClient.addr, targetClientData)
	sendJSON(targetClient.conn, targetClient.addr, srcClientData)
}

func refreshClientInfo(conn *net.UDPConn, addr *net.UDPAddr, clientId string) {
	client := clientMap[clientId]
	if client == nil {
		clientMap[clientId] = &Client{
			addr:         addr,
			conn:         conn,
			lastBeatTime: time.Now().Unix(),
		}
		client = clientMap[clientId]
	} else {
		client.conn = conn
		client.addr = addr
	}
}

func buildNatClientJson(client *Client, clientId string) interface{} {
	glog.Debugf("获取客户端JSON %s", client.addr.String())
	// 确保地址有效
	if client.addr == nil || client.addr.IP == nil {
		return map[string]interface{}{}
	}

	// 提取IPv4字节（兼容IPv4映射的IPv6地址）
	ipv4 := client.addr.IP.To4()
	if ipv4 == nil {
		return map[string]interface{}{}
	}
	// todo recovery
	// if clientId == "75992258" {
	// ipv4 = net.ParseIP("192.168.93.34").To4() //linux
	// } else if clientId == "45173539" {
	// ipv4 = net.ParseIP("192.168.93.10").To4() // windows
	// } else if clientId == "51157139" {
	// ipv4 = net.ParseIP("192.168.0.110").To4() // mac
	// }
	// 构建数据映射（带类型转换）
	return map[string]interface{}{
		"path":     "connectPeer",
		"ip0":      int8(ipv4[0]),    // 第一个字节转为int8
		"ip1":      int8(ipv4[1]),    // 第二个字节转为int8
		"ip2":      int8(ipv4[2]),    // 第三个字节转为int8
		"ip3":      int8(ipv4[3]),    // 第四个字节转为int8
		"port":     client.addr.Port, // todo recovery client.addr.Port,
		"clientId": clientId,
	}
}

func autoRemoveOfflineClient() {
	go func() {
		for {
			now := time.Now().Unix()
			for clientId, client := range clientMap {
				if now-client.lastBeatTime > 30 {
					delete(clientMap, clientId)
					glog.Debugf("客户端 %s 已下线!", clientId)
					session := natSessionMap[clientId]
					if session == nil {
						continue
					}
					// 清理已关闭的NAT会话
					delete(natSessionMap, session.peerTwoId)
					delete(natSessionMap, session.peerOneId)
					notifyPeerDisconnect(session, clientId)
				}
			}
			time.Sleep(time.Second * 1)
		}
	}()
}

func notifyPeerDisconnect(session *NatSession, offlineClientId string) {
	// 禁用中转模式
	disableRelay(session.peerOneId, session.peerTwoId)

	// 通知另一个对等节点关闭连接
	conn := session.peerOneConn
	addr := session.peerOneAddr
	id := session.peerOneId
	if offlineClientId == session.peerOneId {
		conn = session.peerTwoConn
		addr = session.peerTwoAddr
		id = session.peerTwoId
	}
	sendJSON(conn, addr, map[string]interface{}{
		"path": "disconnectPeer",
	})
	glog.Debugf("已通知 %s 与对等节点断开连接", id)
}

func main() {
	// 注册路由处理函数
	RegisterHandler("ping", pingHandler)
	RegisterHandler("notifyChangePort", notifyChangePortHandler)
	RegisterHandler("portChanged", portChangedHandler)

	// 注册中转相关处理函数
	RegisterHandler("enableRelay", enableRelayHandler)

	// 注册中转延迟测试处理函数
	RegisterHandler("relayLatencyTest", relayLatencyTestHandler)
	// 注册中转延迟回复处理函数
	RegisterHandler("relayLatencyReply", relayLatencyReplyHandler)

	// 自动移除离线节点
	autoRemoveOfflineClient()
	// 启动服务器
	startUDPServer(17709)
}
