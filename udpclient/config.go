package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"os"
	"sync"

	"github.com/venshao/natun/glog"
)

// Config 配置结构体
type Config struct {
	PunchHole PunchHoleConfig `json:"punch_hole"`
	Server    ServerConfig    `json:"server"`
	LogLevel  string          `json:"log_level"`
	TunIP     string          `json:"tun_ip"`
	ClientID  string          `json:"client_id"`
	ClientPwd string          `json:"client_pwd"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// PunchHoleConfig 打洞配置
type PunchHoleConfig struct {
	MaxConcurrency int  `json:"max_concurrency"`
	PortRange      int  `json:"port_range"`
	BasePortOffset int  `json:"base_port_offset"`
	EnableRelay    bool `json:"enable_relay"`
	PunchTimeout   int  `json:"punch_timeout"`
	RelayFallback  bool `json:"relay_fallback"`
}

var (
	config     *Config
	configOnce sync.Once
	configFile = "config.json"
)

// createDefaultConfig 创建默认配置
func createDefaultConfig() *Config {
	return &Config{
		PunchHole: PunchHoleConfig{
			MaxConcurrency: 2,
			PortRange:      16,
			BasePortOffset: 0,
			EnableRelay:    true,
			PunchTimeout:   20,
			RelayFallback:  true,
		},
		Server: ServerConfig{
			Host: "117.72.206.26",
			Port: 17709,
		},
		LogLevel:  "INFO",
		TunIP:     generateRandomTunIP(),
		ClientID:  generateRandomClientId(8),
		ClientPwd: generateRandomPwd(6),
	}
}

// ensureConfigDefaults 确保配置字段有默认值
func ensureConfigDefaults(cfg *Config) {
	if cfg.TunIP == "" {
		cfg.TunIP = generateRandomTunIP()
	}
	if cfg.ClientID == "" {
		cfg.ClientID = generateRandomClientId(8)
	}
	if cfg.ClientPwd == "" {
		cfg.ClientPwd = generateRandomPwd(6)
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "117.72.206.26"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 17709
	}
}

// LoadConfig 加载配置文件
func LoadConfig() *Config {
	configOnce.Do(func() {
		// 创建默认配置
		config = createDefaultConfig()

		// 尝试读取配置文件
		data, err := os.ReadFile(configFile)
		if err != nil {
			glog.Warningf("[CONFIG]读取配置文件失败，使用默认配置: %v", err)
			// 保存默认配置文件
			saveConfig(config)
			return
		}

		// 解析JSON配置
		if err := json.Unmarshal(data, config); err != nil {
			glog.Errorf("[CONFIG]解析配置文件失败: %v", err)
			return
		}

		// 确保配置字段有默认值
		ensureConfigDefaults(config)

		glog.Debugf("[CONFIG]成功加载配置文件: %s", configFile)
		glog.Debugf("[CONFIG]打洞配置 - 最大并发数: %d, 端口范围: %d, 基础端口偏移: %d, 启用中转: %v, 打洞超时: %d秒, 自动回退: %v",
			config.PunchHole.MaxConcurrency, config.PunchHole.PortRange, config.PunchHole.BasePortOffset,
			config.PunchHole.EnableRelay, config.PunchHole.PunchTimeout, config.PunchHole.RelayFallback)
		glog.Debugf("[CONFIG]TUN IP: %s, 客户端ID: %s", config.TunIP, config.ClientID)
	})

	return config
}

// saveConfig 保存配置到文件
func saveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		glog.Errorf("[CONFIG]序列化配置失败: %v", err)
		return err
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		glog.Errorf("[CONFIG]保存配置文件失败: %v", err)
		return err
	}

	glog.Debugf("[CONFIG]已保存配置文件: %s", configFile)
	return nil
}

// SaveConfig 保存当前配置到JSON文件
func SaveConfig() error {
	if config == nil {
		return fmt.Errorf("配置未初始化")
	}
	return saveConfig(config)
}

// GetConfig 获取配置实例
func GetConfig() *Config {
	if config == nil {
		return LoadConfig()
	}
	return config
}

// generateRandomTunIP 生成随机TUN IP
func generateRandomTunIP() string {
	// 随机生成TUN设备IP地址
	randomIp := fmt.Sprintf("10.10.10.%d", mathrand.Intn(250)+2)
	return randomIp
}

// generateRandomClientId 生成随机客户端ID
func generateRandomClientId(length int) string {
	randomBytes := getRandomStrAsBytes(length)
	return string(randomBytes)
}

// generateRandomPwd 生成随机密码
func generateRandomPwd(length int) string {
	randomBytes := getRandomStrAsBytes(length)
	return string(randomBytes)
}

// getRandomStrAsBytes 生成随机字符串字节数组
func getRandomStrAsBytes(length int) []byte {
	const charset = "123456789"
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	for i := 0; i < length; i++ {
		randomBytes[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return randomBytes
}
