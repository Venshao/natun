package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/venshao/natun/gjson"
	"github.com/venshao/natun/glog"
)

type Peer struct {
	clientId                    string
	peerAddr                    *net.UDPAddr
	peerVirtualIp               string
	peerAlive                   bool
	latency                     int
	cancelBeatAndTunReadRoutine *context.CancelFunc
}

var peer *Peer = &Peer{
	clientId:                    "",
	peerAddr:                    nil,
	peerAlive:                   false,
	peerVirtualIp:               "",
	latency:                     -1,
	cancelBeatAndTunReadRoutine: nil,
}

// 从配置中获取客户端ID和密码
func getClientId() string {
	return GetConfig().ClientID
}

func getClientPassword() string {
	return GetConfig().ClientPwd
}

func getTunIP() string {
	return GetConfig().TunIP
}

var natConnection = &NatConnection{}

// 当前客户端公网IP
var myPubNetIp string = ""

// 当前客户端公网端口
var myPubNetPort int = 0

// TUN设备
var tun NetDevice = nil

// 第一次收到对方请求需要加锁避免重复开启心跳协程
var mu sync.Mutex

// 通用JSON发送方法（使用gjson序列化）
func call(conn *net.UDPConn, addr *net.UDPAddr, path string, data interface{}) error {
	// 序列化数据
	jsonData := gjson.New(data)
	if jsonData == nil {
		glog.Errorf("[INNER]创建JSON对象失败")
		return fmt.Errorf("[INNER]创建JSON对象失败")
	}

	// 设置路径字段
	err := jsonData.Set("path", path)
	if err != nil {
		return err
	}

	// 转换为字节
	respBytes, err := jsonData.ToJson()
	if err != nil {
		glog.Errorf("[INNER]JSON序列化失败: %v", err)
		return err
	}

	// 发送UDP数据包
	if _, err := conn.WriteToUDP(respBytes, addr); err != nil {
		glog.Errorf("[INNER]发送数据失败: %v", err)
		return err
	}

	return nil
}

// 向对等节点发送心跳 防止连接断开
func beatPeer(conn *net.UDPConn, addr *net.UDPAddr) {
	err := call(conn, addr, "beat", map[string]interface{}{
		// usePort 代表向对等节点打洞使用的目的端口，目的是为了在打洞成功后知道是用的哪个目的端口打洞成功，方便后续基于此端口进行通讯
		"usePort": addr.Port,
		"vip":     getTunIP(),
		"c":       rand.Intn(100000),
		"t":       -1, // 不再在心跳中包含时间戳
		"a":       rand.Intn(100000),
		"id":      getClientId(),
	})
	if err != nil {
		glog.Errorf("[INNER]向对等节点发送心跳失败：%v", err)
	} else {
		glog.Debugf("[INNER]向对等节点%s发送心跳成功", addr.String())
	}
}

func pongHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	glog.Debug("[INNER]收到注册中心的pong")
	myPubNetIp = json.GetString("clientIp")
	myPubNetPort = json.GetInt("clientPort")
}

func beatHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	// 此方法被调用，说明对等节点基于usePort成功打洞到我方，需要将此端口返回给对等节点
	id := json.GetString("id")
	if id == getClientId() {
		glog.Debug("[INNER]收到自己的打洞报文,丢弃!!!")
		return
	}
	usePort := json.GetInt("usePort")
	peer.peerAlive = true

	// 设置直连模式
	cm := GetConnectionManager()
	cm.SetMode(ModeDirect)
	cm.SetConnecting(false, "连接成功")

	// 启动直连模式延迟测试（防止重复启动）
	if peer.cancelBeatAndTunReadRoutine == nil {
		go startDirectLatencyTest(conn)
	}

	glog.Debugf("[INNER]收到对等节点%s的心跳,猜测到的我方port=%d", addr.String(), usePort)
	responseBeatAck(conn, addr, json)
	// 第一次收到对方请求
	if peer.peerAddr == nil {
		mu.Lock()
		defer mu.Unlock()
		if peer.peerAddr == nil {
			atomic.StorePointer(
				(*unsafe.Pointer)(unsafe.Pointer(&peer.peerAddr)),
				unsafe.Pointer(addr),
			)
			peer.peerVirtualIp = json.GetString("vip")
			glog.Debugf("[INNER]peerVirtualIp is %s", peer.peerVirtualIp)
			ctx, cancel := context.WithCancel(context.Background())
			peer.cancelBeatAndTunReadRoutine = &cancel
			// 仅第一次收到心跳时才启动协程进行定时心跳
			go func(ctx context.Context) {
				lastBeatTime := time.Now().Unix()
				for {
					select {
					case <-ctx.Done():
						glog.Debugf("[INNER]向对等节点发心跳的协程退出")
						return
					default:
						if time.Now().Unix()-lastBeatTime < 5 {
							time.Sleep(time.Millisecond * 1)
							continue
						}
						lastBeatTime = time.Now().Unix()
						if peer.peerAddr != nil {
							beatPeer(conn, peer.peerAddr)
						}
					}
				}
			}(ctx)
			NewTunDevice()
			// 启动隧道处理, 从TUN读取并发送到隧道中
			go func(ctx context.Context) {
				packet := make([]byte, 65536)
				for {
					select {
					case <-ctx.Done():
						glog.Debugf("[TUN]TUN读取协程退出")
						return
					default:
						n, err := tun.Read(packet)
						if err != nil {
							errStr := fmt.Sprintf("%v", err)
							if strings.Contains(errStr, "No more data is available") {
								time.Sleep(time.Nanosecond * 1)
							} else {
								glog.Errorf("[TUN]从TUN设备读取失败：%v", err)
							}
							continue
						}

						// 解析IP包并输出调试信息
						if n >= 20 {
							parseAndLogIPPacket(packet[:n], "TUN_READ")
						} else {
							glog.Debugf("[TUN_READ]从TUN设备读出数据%d字节", n)
						}

						sendToTunnel(conn, packet, n)
					}
				}
			}(ctx)
		}
	}
}

func NewTunDevice() {
	if tun == nil {
		tun = CreateTun()
	}
}

func responseBeatAck(conn *net.UDPConn, addr *net.UDPAddr, json *gjson.Json) {
	t := json.GetInt64("t")
	if t > 0 {
		err := call(conn, addr, "beatAck", map[string]interface{}{
			"t": t,
		})
		if err != nil {
			glog.Warningf("[INNER]向对等节点发送心跳ACK失败：%v", err)
		}
	}
}

// beatAckHandler 处理对等节点发来的心跳ack，目的是计算往返延迟
func beatAckHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	t := json.GetInt64("t")
	peer.latency = int(time.Now().UnixMilli() - t)
	glog.Debugf("[INNER]收到心跳Ack,计算往返延迟=%dms", peer.latency)
}

// relayEnabledHandler 处理服务器中转模式开启成功的通知
func relayEnabledHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	// 检查是否为中转模式
	cm := GetConnectionManager()
	if !cm.IsRelayMode() {
		glog.Warning("[INNER]收到中转模式开启通知但当前不是中转模式，忽略")
		return
	}

	peerId := json.GetString("peerId")
	if peerId == "" {
		glog.Warning("[INNER]收到中转模式开启通知但peerId为空")
		return
	}

	// 获取对等节点的虚拟IP
	peerVip := json.GetString("vip")
	if peerVip != "" {
		peer.peerVirtualIp = peerVip
		glog.Debugf("[INNER]中转模式：设置对等节点虚拟IP为 %s", peerVip)
	}

	glog.Infof("[INNER]中转模式已开启，对等节点：%s，虚拟IP：%s", peerId, peerVip)

	// 设置对等节点存活状态
	peer.peerAlive = true

	// 设置连接成功状态
	cm.SetConnecting(false, "连接成功")

	// 启动中转模式延迟测试（防止重复启动）
	if peer.cancelBeatAndTunReadRoutine == nil {
		go startRelayLatencyTest(conn)
	}

	// 第一次收到中转模式开启通知，初始化TUN设备和数据读取
	if peer.cancelBeatAndTunReadRoutine == nil {
		mu.Lock()
		defer mu.Unlock()
		if peer.cancelBeatAndTunReadRoutine == nil {
			ctx, cancel := context.WithCancel(context.Background())
			peer.cancelBeatAndTunReadRoutine = &cancel

			// 初始化TUN设备
			NewTunDevice()

			// 启动隧道处理, 从TUN读取并发送到隧道中
			go func(ctx context.Context) {
				packet := make([]byte, 65536)
				for {
					select {
					case <-ctx.Done():
						glog.Debugf("[TUN]TUN读取协程退出")
						return
					default:
						n, err := tun.Read(packet)
						if err != nil {
							errStr := fmt.Sprintf("%v", err)
							if strings.Contains(errStr, "No more data is available") {
								time.Sleep(time.Nanosecond * 1)
							} else {
								glog.Errorf("[TUN]从TUN设备读取失败：%v", err)
							}
							continue
						}

						// 解析IP包并输出调试信息
						if n >= 20 {
							parseAndLogIPPacket(packet[:n], "TUN_READ")
						} else {
							glog.Debugf("[TUN_READ]从TUN设备读出数据%d字节", n)
						}

						sendToTunnel(conn, packet, n)
					}
				}
			}(ctx)

			glog.Infof("[INNER]中转模式：已初始化TUN设备并启动数据读取协程")
		}
	}
}

// requestConnectPeer 向对等节点发起连接
func requestConnectPeer(targetClientId string, password string) {
	// 先执行changePort
	natConnection.changePort(getRandPort())
	time.Sleep(time.Millisecond * 100)
	// 让服务器通知对方也执行changePort
	err := call(natConnection.listen, serverAddr, "notifyChangePort", map[string]interface{}{
		"srcId":    getClientId(),
		"targetId": targetClientId,
		"tp":       HashMD5(password),
	})
	if err != nil {
		glog.Errorf("[INNER]向服务器发送changePort失败：%v", err)
		return
	}
	glog.Debug("[INNER]已向服务器申请通知对等节点更换端口")
}

func changePortHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	// 如果当前已经有对等节点，则拒绝
	if peer.peerAlive {
		glog.Warningf("[INNER]拒绝changePort请求，当前已有对等节点")
		return
	}
	password := json.GetString("p")
	if password != HashMD5(getClientPassword()) {
		glog.Warningf("[INNER]changePort请求密码错误，拒绝连接")
		return
	}
	natConnection.changePort(getRandPort())
	time.Sleep(time.Millisecond * 100)
	err := call(natConnection.listen, serverAddr, "portChanged", map[string]interface{}{
		"clientId": getClientId(),
	})
	if err != nil {
		glog.Errorf("[INNER]向服务器发送端口已变更的回调失败：%v", err)
	}
}

/**
 * 这个方法由注册中心调用，需要连接的双方会同时执行此方法向对方发起连接
 */
func connectPeerHandler(conn *net.UDPConn, _ *net.UDPAddr, _ string, json *gjson.Json) {
	// 解析基础端口信息
	basePort := json.GetInt("port")
	peer.clientId = json.GetString("clientId")
	ip := net.IPv4(
		byte(json.GetInt8("ip0")),
		byte(json.GetInt8("ip1")),
		byte(json.GetInt8("ip2")),
		byte(json.GetInt8("ip3")),
	)
	if ip.String() == "0.0.0.0" {
		glog.Errorf("[INNER]收到注册中心命令：连接对等节点，但对等IP为0.0.0.0，放弃")
		return
	}

	// 设置对等节点信息
	peerAddr := &net.UDPAddr{IP: ip, Port: basePort}
	GetConnectionManager().SetPeerInfo(peer.clientId, peerAddr)

	// 设置连接状态
	cm := GetConnectionManager()
	cm.SetConnecting(true, "端口探测中...")

	// 从配置文件读取并发控制参数
	cfg := GetConfig()
	maxConcurrency := cfg.PunchHole.MaxConcurrency
	portRange := cfg.PunchHole.PortRange
	basePortOffset := cfg.PunchHole.BasePortOffset

	// 创建带缓冲的channel控制并发
	sem := make(chan struct{}, maxConcurrency)
	done := make(chan struct{})

	// 计算有效端口范围
	startPort := basePort + basePortOffset
	if startPort < 1 {
		startPort = 1
	}

	endPort := startPort + portRange
	if endPort > 65535 {
		endPort = 65535
	}

	// 启动端口扫描协程
	go func() {
		ports := make([]int, endPort-startPort)
		for i := startPort; i < endPort; i++ {
			ports[i-startPort] = i
		}
		// 不断尝试预测端口
		for tryCount := 0; tryCount < 3 && !peer.peerAlive; tryCount++ {
			rand.Shuffle(len(ports), func(i, j int) {
				ports[i], ports[j] = ports[j], ports[i]
			})
			for _, port := range ports {
				// 不要向自己打洞
				if myPubNetPort == port && ip.String() == myPubNetIp {
					continue
				}
				select {
				case <-done:
					return
				case sem <- struct{}{}:
					go func(p int) {
						defer func() { <-sem }()

						targetAddr := &net.UDPAddr{
							IP:   ip,
							Port: p,
						}

						// 发送探测包
						beatPeer(conn, targetAddr)

						// 随机间隔避免洪水攻击
						time.Sleep(time.Duration(10+rand.Intn(50)) * time.Millisecond)
					}(port)
				}
			}
			time.Sleep(time.Second * 1)
		}
		close(done)
		glog.Debug("[INNER]>>>>>>>>>>>>>>>端口探测结束")

		// 检查打洞是否成功，如果失败则切换到中转模式
		cfg := GetConfig()
		if cfg.PunchHole.EnableRelay && cfg.PunchHole.RelayFallback {
			// 等待打洞超时时间，期间保持连接状态
			cm := GetConnectionManager()
			cm.SetConnecting(true, "等待打洞结果...")

			time.Sleep(time.Duration(cfg.PunchHole.PunchTimeout) * time.Second)

			// 检查是否仍然没有直连成功
			if !peer.peerAlive {
				glog.Warningf("[INNER]打洞超时（%d秒），切换到中转模式", cfg.PunchHole.PunchTimeout)
				cm.SetMode(ModeRelay)

				// 通知服务器启用中转模式，同时交换虚拟IP
				err := call(conn, serverAddr, "enableRelay", map[string]interface{}{
					"srcId":    getClientId(),
					"targetId": peer.clientId,
					"vip":      getTunIP(), // 发送自己的虚拟IP
				})
				if err != nil {
					glog.Errorf("[INNER]通知服务器启用中转模式失败：%v", err)
					// 如果中转模式也失败，设置连接失败状态
					cm.SetConnectFailed(true, "连接失败：无法建立直连或中转连接")
				} else {
					glog.Infof("[INNER]已通知服务器启用中转模式")
				}
			}
		}
	}()
}

func disconnectPeerHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	glog.Debugf("[INNER]收到注册中心命令：断开对等节点")
	peer.peerAlive = false

	// 设置断开模式
	cm := GetConnectionManager()
	cm.SetMode(ModeDisconnected)
	cm.SetConnecting(false, "连接已断开")
	if peer.cancelBeatAndTunReadRoutine != nil {
		// 取消向对等节点发心跳的协程和TUN设备数据读取的协程
		(*peer.cancelBeatAndTunReadRoutine)()
		peer.cancelBeatAndTunReadRoutine = nil
		glog.Debug("[TUN]已关闭对等节点心跳协程和TUN设备读取协程")
	}
	// 关闭TUN设备
	if tun != nil {
		err := tun.Close()
		if err != nil {
			glog.Errorf("[TUN]关闭TUN设备失败：%v", err)
		}
		tun = nil
		glog.Debug("[TUN]已关闭TUN设备")
	}
	peer.clientId = ""
	peer.peerAddr = nil
	peer.peerAlive = false
	peer.peerVirtualIp = ""
	peer.latency = -1
	peer.cancelBeatAndTunReadRoutine = nil
}

func getRandPort() int {
	// todo recovery
	clientPort := rand.Intn(65536-10000) + 10000
	//clientPort := 10000
	return clientPort
}

func initClient() {
	// 加载配置文件
	GetConfig()

	// 初始化服务器地址
	initServerAddr()

	// 当前客户端端口 10000-65535
	clientPort := getRandPort()
	glog.Debugf("[INNER]本机clientId=%s", getClientId())
	glog.Debugf("[INNER]本机端口号=%d", clientPort)
	// 接收注册中心返回的pong响应
	natConnection.RegisterResponseHandler("pong", pongHandler)
	// 接收注册中心发送的changePort命令
	natConnection.RegisterResponseHandler("changePort", changePortHandler)
	// 接收注册中心发送的连接对等节点的命令
	natConnection.RegisterResponseHandler("connectPeer", connectPeerHandler)
	// 接收注册中心发送的断开对等节点的命令
	natConnection.RegisterResponseHandler("disconnectPeer", disconnectPeerHandler)

	// 接收对等节点的心跳
	natConnection.RegisterResponseHandler("beat", beatHandler)
	// 接收对等节点的心跳ACK 计算往返延迟
	natConnection.RegisterResponseHandler("beatAck", beatAckHandler)

	// 接收服务器中转模式开启成功的通知
	natConnection.RegisterResponseHandler("relayEnabled", relayEnabledHandler)

	// 接收中转模式延迟测试
	natConnection.RegisterResponseHandler("relayLatencyTest", relayLatencyTestHandler)
	// 接收中转模式延迟测试回复
	natConnection.RegisterResponseHandler("relayLatencyReply", relayLatencyReplyHandler)

	natConnection.StartClient(clientPort)
}

func main() {
	if !IsAdmin() {
		glog.Warning("请以管理员/root身份运行...")
		fmt.Scanln()
		return
	}
	cfg := GetConfig()
	glog.SetLevelString(cfg.LogLevel)
	initClient()
	startWebServer()
}

// startDirectLatencyTest 启动直连模式延迟测试
func startDirectLatencyTest(conn *net.UDPConn) {
	// 异步等待2秒后发送第一个测试包
	go func() {
		time.Sleep(2 * time.Second)

		// 检查是否仍然在直连模式
		cm := GetConnectionManager()
		if !cm.IsDirectMode() {
			glog.Debug("[INNER]直连模式已结束，停止延迟测试")
			return
		}

		// 发送第一个测试包
		sendDirectLatencyTest(conn)
	}()

	// 每15秒发送一次延迟测试包
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 检查是否仍然在直连模式
		cm := GetConnectionManager()
		if !cm.IsDirectMode() {
			glog.Debug("[INNER]直连模式已结束，停止延迟测试")
			return
		}

		// 发送延迟测试包
		sendDirectLatencyTest(conn)
	}
}

// sendDirectLatencyTest 发送直连模式延迟测试包
func sendDirectLatencyTest(conn *net.UDPConn) {
	// 获取对等节点信息
	peerClientId, peerAddr := GetConnectionManager().GetPeerInfo()
	if peerClientId == "" || peerAddr == nil {
		glog.Warning("[INNER]直连模式延迟测试：无法获取对等节点信息")
		return
	}

	// 发送延迟测试包
	timestamp := time.Now().UnixMilli()
	err := call(conn, peerAddr, "beat", map[string]interface{}{
		"usePort": peerAddr.Port,
		"vip":     getTunIP(),
		"c":       rand.Intn(100000),
		"t":       timestamp, // 带时间戳的延迟测试包
		"a":       rand.Intn(100000),
		"id":      getClientId(),
	})
	if err != nil {
		glog.Errorf("[INNER]直连模式延迟测试：发送失败：%v", err)
	} else {
		glog.Debugf("[INNER]直连模式延迟测试：已发送测试包，时间戳：%d", timestamp)
	}
}

// startRelayLatencyTest 启动中转模式延迟测试
func startRelayLatencyTest(conn *net.UDPConn) {
	// 异步等待2秒后发送第一个测试包
	go func() {
		time.Sleep(2 * time.Second)

		// 检查是否仍然在中转模式
		cm := GetConnectionManager()
		if !cm.IsRelayMode() {
			glog.Debug("[INNER]中转模式已结束，停止延迟测试")
			return
		}

		// 发送第一个测试包
		sendRelayLatencyTest(conn)
	}()

	// 每15秒发送一次延迟测试包
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 检查是否仍然在中转模式
		cm := GetConnectionManager()
		if !cm.IsRelayMode() {
			glog.Debug("[INNER]中转模式已结束，停止延迟测试")
			return
		}

		// 发送延迟测试包
		sendRelayLatencyTest(conn)
	}
}

// sendRelayLatencyTest 发送中转模式延迟测试包
func sendRelayLatencyTest(conn *net.UDPConn) {
	// 获取对等节点信息
	peerClientId, _ := GetConnectionManager().GetPeerInfo()
	if peerClientId == "" {
		glog.Warning("[INNER]中转模式延迟测试：无法获取对等节点ID")
		return
	}

	// 发送延迟测试包
	timestamp := time.Now().UnixMilli()
	err := call(conn, serverAddr, "relayLatencyTest", map[string]interface{}{
		"targetId":  peerClientId,
		"timestamp": timestamp,
	})
	if err != nil {
		glog.Errorf("[INNER]中转模式延迟测试：发送失败：%v", err)
	} else {
		glog.Debugf("[INNER]中转模式延迟测试：已发送测试包，时间戳：%d", timestamp)
	}
}

// relayLatencyTestHandler 处理中转模式延迟测试
func relayLatencyTestHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	// 检查是否为中转模式
	cm := GetConnectionManager()
	if !cm.IsRelayMode() {
		glog.Warning("[INNER]收到中转延迟测试但当前不是中转模式，忽略")
		return
	}

	timestamp := json.GetInt64("timestamp")
	if timestamp > 0 {
		// 立即回复延迟测试包
		peerClientId, _ := GetConnectionManager().GetPeerInfo()
		if peerClientId != "" {
			err := call(conn, serverAddr, "relayLatencyReply", map[string]interface{}{
				"targetId":  peerClientId,
				"timestamp": timestamp, // 回复相同的时间戳
			})
			if err != nil {
				glog.Errorf("[INNER]中转模式延迟测试：回复失败：%v", err)
			} else {
				glog.Debugf("[INNER]中转模式延迟测试：已回复测试包，时间戳：%d", timestamp)
			}
		}
	}
}

// relayLatencyReplyHandler 处理中转模式延迟测试回复
func relayLatencyReplyHandler(conn *net.UDPConn, addr *net.UDPAddr, path string, json *gjson.Json) {
	// 检查是否为中转模式
	cm := GetConnectionManager()
	if !cm.IsRelayMode() {
		glog.Warning("[INNER]收到中转延迟回复但当前不是中转模式，忽略")
		return
	}

	timestamp := json.GetInt64("timestamp")
	if timestamp > 0 {
		// 计算往返延迟
		peer.latency = int(time.Now().UnixMilli() - timestamp)
		glog.Debugf("[INNER]中转模式延迟测试：往返延迟=%dms", peer.latency)
	}
}
