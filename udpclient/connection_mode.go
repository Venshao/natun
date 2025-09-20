package main

import (
	"net"
	"sync"
	"time"

	"github.com/venshao/natun/glog"
)

// ConnectionMode 连接模式
type ConnectionMode int

const (
	ModeDirect       ConnectionMode = iota // 直连模式（打洞成功）
	ModeRelay                              // 中转模式（通过服务器转发）
	ModeDisconnected                       // 断开状态
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	mode           ConnectionMode
	peerAddr       *net.UDPAddr
	peerClientId   string
	lastModeChange time.Time
	isConnecting   bool
	connectFailed  bool
	connectMessage string
	mu             sync.RWMutex
}

var connectionManager = &ConnectionManager{
	mode:           ModeDisconnected,
	peerAddr:       nil,
	peerClientId:   "",
	lastModeChange: time.Time{},
}

// GetMode 获取当前连接模式
func (cm *ConnectionManager) GetMode() ConnectionMode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.mode
}

// SetMode 设置连接模式
func (cm *ConnectionManager) SetMode(mode ConnectionMode) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldMode := cm.mode
	cm.mode = mode
	cm.lastModeChange = time.Now()

	if oldMode != mode {
		glog.Infof("[CONNECTION]连接模式从 %s 切换到 %s", getModeString(oldMode), getModeString(mode))
	}
}

// SetPeerInfo 设置对等节点信息
func (cm *ConnectionManager) SetPeerInfo(clientId string, addr *net.UDPAddr) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.peerClientId = clientId
	cm.peerAddr = addr
}

// GetPeerInfo 获取对等节点信息
func (cm *ConnectionManager) GetPeerInfo() (string, *net.UDPAddr) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.peerClientId, cm.peerAddr
}

// IsDirectMode 是否为直连模式
func (cm *ConnectionManager) IsDirectMode() bool {
	return cm.GetMode() == ModeDirect
}

// IsRelayMode 是否为中转模式
func (cm *ConnectionManager) IsRelayMode() bool {
	return cm.GetMode() == ModeRelay
}

// IsConnected 是否已连接
func (cm *ConnectionManager) IsConnected() bool {
	mode := cm.GetMode()
	return mode == ModeDirect || mode == ModeRelay
}

// getModeString 获取模式字符串
func getModeString(mode ConnectionMode) string {
	switch mode {
	case ModeDirect:
		return "直连模式"
	case ModeRelay:
		return "中转模式"
	case ModeDisconnected:
		return "断开状态"
	default:
		return "未知模式"
	}
}

// SetConnecting 设置连接状态
func (cm *ConnectionManager) SetConnecting(connecting bool, message string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.isConnecting = connecting
	cm.connectMessage = message
	cm.connectFailed = false
}

// SetConnectFailed 设置连接失败
func (cm *ConnectionManager) SetConnectFailed(failed bool, message string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connectFailed = failed
	cm.connectMessage = message
	cm.isConnecting = false
}

// IsConnecting 是否正在连接
func (cm *ConnectionManager) IsConnecting() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isConnecting
}

// IsConnectFailed 连接是否失败
func (cm *ConnectionManager) IsConnectFailed() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connectFailed
}

// GetConnectMessage 获取连接消息
func (cm *ConnectionManager) GetConnectMessage() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connectMessage
}

// GetConnectionManager 获取连接管理器实例
func GetConnectionManager() *ConnectionManager {
	return connectionManager
}
