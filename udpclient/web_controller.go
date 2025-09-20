package main

import (
	"embed"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/venshao/natun/gin"
	"github.com/venshao/natun/glog"
)

//go:embed static/*
var staticFiles embed.FS

// DeviceInfo 设备信息结构体
type DeviceInfo struct {
	ClientId string `json:"clientId"`
	IP       string `json:"IP"`
	Alive    bool   `json:"alive"`
	NatType  string `json:"natType"`
	Latency  int    `json:"latency"`
	Password string `json:"password"`
}

// ConnectionStatus 连接状态信息
type ConnectionStatus struct {
	Mode           string `json:"mode"`           // 连接模式：直连模式、中转模式、断开状态
	ModeCode       int    `json:"modeCode"`       // 模式代码：0=直连，1=中转，2=断开
	IsConnected    bool   `json:"isConnected"`    // 是否已连接
	StatusText     string `json:"statusText"`     // 状态文本
	IsConnecting   bool   `json:"isConnecting"`   // 是否正在连接中
	ConnectFailed  bool   `json:"connectFailed"`  // 连接是否失败
	ConnectMessage string `json:"connectMessage"` // 连接状态消息
}

// ConnectRequest 连接请求结构体
type ConnectRequest struct {
	TargetId  string `json:"targetId"`
	TargetPwd string `json:"targetPwd"`
}

// ResetPasswordRequest 重设密码请求结构体
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

// 获取本机设备信息
func getDeviceHandler(c *gin.Context) {
	deviceInfo := DeviceInfo{
		ClientId: getClientId(),
		IP:       getTunIP(),
		NatType:  "NAT3",
		Password: getClientPassword(),
	}
	c.JSON(http.StatusOK, deviceInfo)
}

// 连接目标设备
func connPeerHandler(c *gin.Context) {
	var req ConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的请求参数"})
		return
	}
	if req.TargetId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "目标识别码不能为空"})
		return
	}
	if len(req.TargetId) != 8 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "目标识别码错误"})
		return
	}
	if req.TargetPwd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "密码不能为空"})
		return
	}

	// 设置连接状态
	GetConnectionManager().SetConnecting(true, "正在连接...")

	requestConnectPeer(req.TargetId, req.TargetPwd)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "正在连接..."})
}

func peerStatusHandler(c *gin.Context) {
	// 获取连接管理器
	cm := GetConnectionManager()
	mode := cm.GetMode()

	// 构建设备信息
	deviceInfo := DeviceInfo{
		ClientId: peer.clientId,
		IP:       peer.peerVirtualIp,
		Alive:    peer.peerAlive,
		Latency:  peer.latency,
	}

	// 构建连接状态信息
	connectionStatus := ConnectionStatus{
		ModeCode:       int(mode),
		IsConnected:    cm.IsConnected(),
		IsConnecting:   cm.IsConnecting(),
		ConnectFailed:  cm.IsConnectFailed(),
		ConnectMessage: cm.GetConnectMessage(),
	}

	// 设置模式文本和状态文本
	switch mode {
	case ModeDirect:
		connectionStatus.Mode = "直连模式"
		connectionStatus.StatusText = "P2P直连"
	case ModeRelay:
		connectionStatus.Mode = "中转模式"
		connectionStatus.StatusText = "服务器中转"
	case ModeDisconnected:
		connectionStatus.Mode = "断开状态"
		if connectionStatus.IsConnecting {
			connectionStatus.StatusText = "正在连接..."
		} else if connectionStatus.ConnectFailed {
			connectionStatus.StatusText = "连接失败"
		} else {
			connectionStatus.StatusText = "未连接"
		}
	default:
		connectionStatus.Mode = "未知模式"
		connectionStatus.StatusText = "未知状态"
	}

	// 返回包含连接状态的响应
	response := map[string]interface{}{
		"device": deviceInfo,
		"status": connectionStatus,
	}

	c.JSON(http.StatusOK, response)
}

// 重设密码
func resetPasswordHandler(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的请求参数"})
		return
	}

	if len(req.NewPassword) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "密码必须为6位数字"})
		return
	}

	// 验证密码是否只包含数字
	for _, ch := range req.NewPassword {
		if ch < '0' || ch > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "密码只能包含数字"})
			return
		}
	}

	// 更新密码
	cfg := GetConfig()
	cfg.ClientPwd = req.NewPassword
	err := SaveConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "密码保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "密码修改成功",
		"password": req.NewPassword,
	})
}

// 打开浏览器函数
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin": // macOS
		cmd = "open"
	default: // linux
		cmd = "xdg-open"
	}
	args = append(args, url)

	if err := exec.Command(cmd, args...).Start(); err != nil {
		glog.Errorf("自动打开浏览器失败，请使用浏览器访问http://127.0.0.1:8898/\n")
	}
	glog.Infof("如果浏览器没有自动打开，请使用浏览器访问http://127.0.0.1:8898/\n")
	glog.Infof("如果浏览器没有自动打开，请使用浏览器访问http://127.0.0.1:8898/\n")
	glog.Infof("如果浏览器没有自动打开，请使用浏览器访问http://127.0.0.1:8898/\n")
}

// 设置缓存控制头部
func setNoCacheHeaders(c *gin.Context) {
	c.W.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.W.Header().Set("Pragma", "no-cache")
	c.W.Header().Set("Expires", "0")
}

// 启动 Web 服务器
func startWebServer() {
	r := gin.New()

	// 将嵌入的文件作为静态资源
	r.StaticFS("/static", staticFiles)

	r.GET("/", func(c *gin.Context) {
		setNoCacheHeaders(c)
		data, _ := staticFiles.ReadFile("static/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	r.GET("/js/vue.global.prod.js", func(c *gin.Context) {
		setNoCacheHeaders(c)
		data, _ := staticFiles.ReadFile("static/js/vue.global.prod.js")
		c.Data(http.StatusOK, "text/javascript; charset=utf-8", data)
	})

	r.GET("/css/style.css", func(c *gin.Context) {
		setNoCacheHeaders(c)
		data, _ := staticFiles.ReadFile("static/css/style.css")
		c.Data(http.StatusOK, "text/css; charset=utf-8", data)
	})

	r.GET("/js/app.js", func(c *gin.Context) {
		setNoCacheHeaders(c)
		data, _ := staticFiles.ReadFile("static/js/app.js")
		c.Data(http.StatusOK, "text/javascript; charset=utf-8", data)
	})

	api := r.Group("/api")
	{
		api.GET("/device", func(c *gin.Context) {
			setNoCacheHeaders(c)
			getDeviceHandler(c)
		})
		api.POST("/connect", func(c *gin.Context) {
			setNoCacheHeaders(c)
			connPeerHandler(c)
		})
		api.GET("/peerStatus", func(c *gin.Context) {
			setNoCacheHeaders(c)
			peerStatusHandler(c)
		})
		api.POST("/resetPassword", func(c *gin.Context) {
			setNoCacheHeaders(c)
			resetPasswordHandler(c)
		})
	}

	openBrowser("http://127.0.0.1:8898")

	err := r.Run("0.0.0.0:8898")
	if err != nil {
		panic(err)
	}
}
