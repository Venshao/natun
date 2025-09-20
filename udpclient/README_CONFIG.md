# 配置文件说明

## config.json

程序启动时会自动读取 `config.json` 配置文件。如果配置文件不存在，程序会自动创建一个包含默认值的配置文件。

### 配置项说明

```json
{
  "punch_hole": {
    "max_concurrency": 2,         // 打洞最大并发数
    "port_range": 16,             // 打洞端口范围
    "base_port_offset": 0,        // 基础端口偏移量
    "enable_relay": true,         // 是否启用中转模式
    "punch_timeout": 20,          // 打洞超时时间（秒）
    "relay_fallback": true        // 超过打洞超时时间自动认为打洞失败，自动切换到中转
  },
  "server": {
    "host": "117.72.206.26",     // 中转服务器IP地址
    "port": 17709                 // 中转服务器端口
  },
  "log_level": "INFO",           // 日志级别
  "tun_ip": "10.10.10.6",       // TUN设备IP地址（虚拟局域网本机IP）
  "client_id": "66668888",      // 客户端唯一标识
  "client_pwd": "123456"        // 客户端密码
}
```

### 参数说明

#### punch_hole 打洞配置
- `max_concurrency`: 打洞时的最大并发数，默认为 2
- `port_range`: 打洞时扫描的端口范围，默认为 16
- `base_port_offset`: 基础端口偏移量，用于计算起始扫描端口，默认为 0
- `enable_relay`: 是否启用中转模式，默认为 true
- `punch_timeout`: 打洞超时时间（秒），默认为 20
- `relay_fallback`: 打洞失败后自动切换到中转模式，默认为 true

#### server 服务器配置
- `host`: 中转服务器IP地址，默认为 "117.72.206.26"
- `port`: 中转服务器端口，默认为 17709

#### 其他配置
- `log_level`: 日志级别，可选值：DEBUG、INFO、WARN、ERROR，默认为 INFO
- `tun_ip`: TUN设备IP地址，格式为 10.10.10.x，程序会自动生成
- `client_id`: 客户端唯一标识，用于区分不同客户端，程序会自动生成
- `client_pwd`: 客户端密码，用于连接验证，程序会自动生成

### 使用示例

1. **基础配置**（推荐新手使用）：
```json
{
  "punch_hole": {
    "max_concurrency": 2,
    "port_range": 16,
    "base_port_offset": 0,
    "enable_relay": true,
    "punch_timeout": 20,
    "relay_fallback": true
  },
  "log_level": "INFO"
}
```

2. **高性能配置**（网络环境良好时）：
```json
{
  "punch_hole": {
    "max_concurrency": 10,
    "port_range": 50,
    "base_port_offset": 0,
    "enable_relay": true,
    "punch_timeout": 15,
    "relay_fallback": true
  },
  "log_level": "WARN"
}
```

3. **低延迟配置**（禁用中转模式）：
```json
{
  "punch_hole": {
    "max_concurrency": 5,
    "port_range": 32,
    "base_port_offset": 0,
    "enable_relay": false,
    "punch_timeout": 30,
    "relay_fallback": false
  },
  "log_level": "INFO"
}
```

4. **调试配置**（开发调试时）：
```json
{
  "punch_hole": {
    "max_concurrency": 1,
    "port_range": 16,
    "base_port_offset": 0,
    "enable_relay": true,
    "punch_timeout": 20,
    "relay_fallback": true
  },
  "log_level": "DEBUG",
  "tun_ip": "10.10.10.100",
  "client_id": "debug001",
  "client_pwd": "123456"
}
```

5. **自定义客户端标识**：
```json
{
  "punch_hole": {
    "max_concurrency": 2,
    "port_range": 16,
    "base_port_offset": 0,
    "enable_relay": true,
    "punch_timeout": 20,
    "relay_fallback": true
  },
  "log_level": "INFO",
  "tun_ip": "10.10.10.x",  // x 可以修改为2~255的数字
  "client_id": "set_your_client_id",
  "client_pwd": "set_your_password"
}
```

6. **私有服务器部署**：
```json
{
  "punch_hole": {
    "max_concurrency": 5,
    "port_range": 32,
    "base_port_offset": 0,
    "enable_relay": true,
    "punch_timeout": 15,
    "relay_fallback": true
  },
  "server": {
    "host": "your-server-ip",  // 替换为您的服务器IP
    "port": 17709              // 替换为您的服务器端口
  },
  "log_level": "INFO"
}
```

### 连接模式说明

程序支持两种连接模式：

1. **直连模式（Direct Mode）**：打洞成功后，客户端之间直接通信，延迟最低
2. **中转模式（Relay Mode）**：打洞失败后，通过服务器转发数据，确保连接可用

### 注意事项

#### 配置管理
- 修改配置文件后需要重启程序才能生效
- 程序会自动生成 `tun_ip`、`client_id`、`client_pwd` 等字段的默认值
- 可以通过Web界面动态修改密码，修改后会自动保存到配置文件
- 建议备份配置文件，避免意外丢失

#### 网络配置
- 建议根据网络环境调整打洞参数
- 过高的并发数可能导致网络拥塞
- 过大的端口范围会增加扫描时间
- 中转模式会增加服务器负载，但能确保连接可用性
- 打洞超时时间不宜设置过短，建议至少20秒

#### 客户端标识
- `client_id` 用于区分不同客户端，建议使用有意义的标识
- `client_pwd` 用于连接验证，建议使用强密码
- `tun_ip` 必须是 10.10.10.x 格式，避免与其他网络冲突

#### 日志配置
- DEBUG 级别会输出详细的调试信息，适合开发调试
- INFO 级别输出一般信息，适合日常使用
- WARN/ERROR 级别只输出警告和错误，适合生产环境

### 配置字段快速参考

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `punch_hole.max_concurrency` | int | 2 | 打洞最大并发数 |
| `punch_hole.port_range` | int | 16 | 打洞端口范围 |
| `punch_hole.base_port_offset` | int | 0 | 基础端口偏移量 |
| `punch_hole.enable_relay` | bool | true | 是否启用中转模式 |
| `punch_hole.punch_timeout` | int | 20 | 打洞超时时间（秒） |
| `punch_hole.relay_fallback` | bool | true | 打洞失败后自动切换到中转 |
| `server.host` | string | "117.72.206.26" | 中转服务器IP地址 |
| `server.port` | int | 17709 | 中转服务器端口 |
| `log_level` | string | "INFO" | 日志级别 |
| `tun_ip` | string | 自动生成 | TUN设备IP地址 |
| `client_id` | string | 自动生成 | 客户端唯一标识 |
| `client_pwd` | string | 自动生成 | 客户端密码 |
