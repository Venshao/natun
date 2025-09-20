package main

import (
	"net"

	"github.com/venshao/natun/gjson"
	"github.com/venshao/natun/glog"
)

// RelaySession 中转会话
type RelaySession struct {
	clientOneId string
	clientTwoId string
	enabled     bool
}

// 中转会话映射
var relaySessions = make(map[string]*RelaySession)

// 客户端虚拟IP映射
var clientVips = make(map[string]string)

// enableRelayHandler 启用中转模式
func enableRelayHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	srcId := json.GetString("srcId")
	targetId := json.GetString("targetId")
	srcVip := json.GetString("vip") // 获取源客户端的虚拟IP

	if srcId == "" || targetId == "" {
		glog.Warningf("[RELAY]启用中转模式失败：缺少必要参数")
		return
	}

	// 保存源客户端的虚拟IP
	clientVips[srcId] = srcVip

	// 创建或更新中转会话
	sessionKey := getSessionKey(srcId, targetId)
	relaySessions[sessionKey] = &RelaySession{
		clientOneId: srcId,
		clientTwoId: targetId,
		enabled:     true,
	}

	glog.Infof("[RELAY]已启用中转模式：%s <-> %s，源客户端虚拟IP：%s", srcId, targetId, srcVip)

	// 检查是否两个客户端都已注册虚拟IP
	targetVip, targetExists := clientVips[targetId]
	if targetExists {
		// 两个客户端都已注册，可以交换虚拟IP
		notifyRelayEnabled(srcId, targetId, targetVip) // 源客户端收到目标客户端的虚拟IP
		notifyRelayEnabled(targetId, srcId, srcVip)    // 目标客户端收到源客户端的虚拟IP
		glog.Infof("[RELAY]虚拟IP交换完成：%s(%s) <-> %s(%s)", srcId, srcVip, targetId, targetVip)
	} else {
		// 目标客户端还未注册，等待目标客户端注册
		glog.Debugf("[RELAY]等待目标客户端 %s 注册虚拟IP", targetId)
	}
}

// relayLatencyTestHandler 处理中转模式延迟测试
func relayLatencyTestHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	targetId := json.GetString("targetId")
	timestamp := json.GetInt64("timestamp")

	if targetId == "" {
		glog.Warningf("[RELAY]中转延迟测试失败：缺少targetId参数")
		return
	}

	// 获取发送方ID
	srcId := getClientIdByAddr(addr)
	if srcId == "" {
		glog.Warningf("[RELAY]无法识别发送方：%s", addr.String())
		return
	}

	// 检查中转会话是否存在
	sessionKey := getSessionKey(srcId, targetId)
	session := relaySessions[sessionKey]
	if session == nil || !session.enabled {
		glog.Warningf("[RELAY]中转会话不存在或未启用：%s -> %s", srcId, targetId)
		return
	}

	// 获取目标客户端
	targetClient := clientMap[targetId]
	if targetClient == nil {
		glog.Warningf("[RELAY]目标客户端不存在：%s", targetId)
		return
	}

	// 转发延迟测试包
	sendJSON(targetClient.conn, targetClient.addr, map[string]interface{}{
		"path":      "relayLatencyTest",
		"timestamp": timestamp,
	})

	glog.Debugf("[RELAY]转发延迟测试包：%s -> %s，时间戳：%d", srcId, targetId, timestamp)
}

// relayLatencyReplyHandler 处理中转模式延迟测试回复
func relayLatencyReplyHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	targetId := json.GetString("targetId")
	timestamp := json.GetInt64("timestamp")

	if targetId == "" {
		glog.Warningf("[RELAY]中转延迟回复失败：缺少targetId参数")
		return
	}

	// 获取发送方ID
	srcId := getClientIdByAddr(addr)
	if srcId == "" {
		glog.Warningf("[RELAY]无法识别发送方：%s", addr.String())
		return
	}

	// 检查中转会话是否存在
	sessionKey := getSessionKey(srcId, targetId)
	session := relaySessions[sessionKey]
	if session == nil || !session.enabled {
		glog.Warningf("[RELAY]中转会话不存在或未启用：%s -> %s", srcId, targetId)
		return
	}

	// 获取目标客户端
	targetClient := clientMap[targetId]
	if targetClient == nil {
		glog.Warningf("[RELAY]目标客户端不存在：%s", targetId)
		return
	}

	// 转发延迟回复包
	sendJSON(targetClient.conn, targetClient.addr, map[string]interface{}{
		"path":      "relayLatencyReply",
		"timestamp": timestamp,
	})

	glog.Debugf("[RELAY]转发延迟回复包：%s -> %s，时间戳：%d", srcId, targetId, timestamp)
}

// getSessionKey 获取会话键
func getSessionKey(id1, id2 string) string {
	if id1 < id2 {
		return id1 + "_" + id2
	}
	return id2 + "_" + id1
}

// getClientIdByAddr 根据地址获取客户端ID
func getClientIdByAddr(addr *net.UDPAddr) string {
	for clientId, client := range clientMap {
		if client.addr.String() == addr.String() {
			return clientId
		}
	}
	return ""
}

// notifyRelayEnabled 通知客户端中转模式已启用
func notifyRelayEnabled(clientId string, peerId string, peerVip string) {
	client := clientMap[clientId]
	if client == nil {
		glog.Warningf("[RELAY]客户端不存在：%s", clientId)
		return
	}

	sendJSON(client.conn, client.addr, map[string]interface{}{
		"path":   "relayEnabled",
		"peerId": peerId,
		"vip":    peerVip, // 发送对等节点的虚拟IP
	})

	glog.Debugf("[RELAY]已通知客户端 %s 中转模式已启用，对等节点虚拟IP：%s", clientId, peerVip)
}

// disableRelay 禁用中转模式
func disableRelay(srcId, targetId string) {
	sessionKey := getSessionKey(srcId, targetId)
	delete(relaySessions, sessionKey)
	glog.Infof("[RELAY]已禁用中转模式：%s <-> %s", srcId, targetId)
}
