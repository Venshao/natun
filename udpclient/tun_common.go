package main

import (
	"fmt"
	"net"

	"github.com/venshao/natun/glog"
)

func sendToTunnel(conn *net.UDPConn, frame []byte, size int) {
	if tun == nil {
		glog.Warning("[TUN]警告：TUN设备为空！无法发送数据")
		return
	}

	// 获取连接管理器
	cm := GetConnectionManager()

	// 直接发送原始TUN数据，让sendDirectPacket根据模式添加协议头
	sendDirectPacket(conn, frame[:size], cm)
}

// HandleReceivedPacket  收到报文后的处理
func HandleReceivedPacket(listen *net.UDPConn, packet []byte) {
	if tun == nil {
		glog.Warning("[TUN]警告：TUN设备为空！无法写入数据")
		return
	}

	// 解析IP包并输出调试信息
	parseAndLogIPPacket(packet, "TUN")

	// 检查数据包长度是否与IP头部声明的长度一致
	if len(packet) >= 20 {
		declaredLength := int(packet[2])<<8 | int(packet[3])
		if len(packet) != declaredLength {
			glog.Warningf("[TUN]数据包长度不匹配: 实际=%d, 声明=%d", len(packet), declaredLength)
		}
	}

	_, err := tun.Write(packet)
	if err != nil {
		glog.Errorf("[TUN]写入TUN设备失败：%v", err)
		return
	}
	glog.Debugf("[TUN]已写入TUN设备%d字节", len(packet))
}

// 解析IP包并输出调试信息
func parseAndLogIPPacket(packet []byte, prefix string) {
	if len(packet) >= 20 {
		totalLength := int(packet[2])<<8 | int(packet[3])
		protocol := packet[9]
		srcIP := net.IP(packet[12:16]).String()
		dstIP := net.IP(packet[16:20]).String()

		glog.Debugf("[%s]收到数据包: %s -> %s, 协议=%d, 长度=%d", prefix, srcIP, dstIP, protocol, totalLength)

		// 打印数据包前32字节的十六进制，用于调试
		hexStr := ""
		for i := 0; i < 32 && i < len(packet); i++ {
			hexStr += fmt.Sprintf("%02x ", packet[i])
		}
		glog.Debugf("[%s]数据包前32字节: %s", prefix, hexStr)

		// 特殊处理ICMP包
		if protocol == 1 { // ICMP
			if len(packet) >= 28 {
				icmpType := packet[20]
				icmpCode := packet[21]
				var icmpTypeStr string
				switch icmpType {
				case 0:
					icmpTypeStr = "Echo Reply"
				case 8:
					icmpTypeStr = "Echo Request"
				case 3:
					icmpTypeStr = "Destination Unreachable"
				case 11:
					icmpTypeStr = "Time Exceeded"
				default:
					icmpTypeStr = fmt.Sprintf("Unknown(%d)", icmpType)
				}
				glog.Debugf("[%s]ICMP包: %s -> %s, 类型=%s, 代码=%d", prefix, srcIP, dstIP, icmpTypeStr, icmpCode)
			}
		}

		// 特殊处理TCP包
		if protocol == 6 && len(packet) >= 40 { // TCP
			srcPort := int(packet[20])<<8 | int(packet[21])
			dstPort := int(packet[22])<<8 | int(packet[23])
			flags := packet[33]

			var flagStr string
			if flags&0x02 != 0 {
				flagStr += "SYN "
			}
			if flags&0x10 != 0 {
				flagStr += "ACK "
			}
			if flags&0x01 != 0 {
				flagStr += "FIN "
			}
			if flags&0x08 != 0 {
				flagStr += "PSH "
			}
			if flags&0x04 != 0 {
				flagStr += "RST "
			}

			glog.Debugf("[%s]TCP包: %s:%d -> %s:%d, 标志=%s", prefix, srcIP, srcPort, dstIP, dstPort, flagStr)
		}

		// 特殊处理UDP包
		if protocol == 17 && len(packet) >= 28 { // UDP
			srcPort := int(packet[20])<<8 | int(packet[21])
			dstPort := int(packet[22])<<8 | int(packet[23])
			glog.Debugf("[%s]UDP包: %s:%d -> %s:%d", prefix, srcIP, srcPort, dstIP, dstPort)
		}
	}
}

// 直接发送数据包（不分包）
func sendDirectPacket(conn *net.UDPConn, tunData []byte, cm *ConnectionManager) {
	if cm.IsDirectMode() {
		// 直连模式：添加直连协议头并发送到对等节点
		if peer.peerAddr != nil {
			// 协议格式: [魔数4B] + [0x01] + [数据长度(2B)] + [TUN数据]
			header := []byte{0x12, 0x34, 0x56, 0x78, 0x01, byte(len(tunData) >> 8), byte(len(tunData) & 0xFF)}
			tunnelPacket := append(header, tunData...)

			//glog.Debugf("[TUN]直连模式：向peer %s 发送包%d字节", peer.peerAddr.String(), len(tunnelPacket))
			conn.WriteToUDP(tunnelPacket, peer.peerAddr)
		} else {
			glog.Warning("[TUN]直连模式：无法向peer发送包,peer.peerAddr为空")
		}
	} else if cm.IsRelayMode() {
		// 中转模式：添加中转协议头并通过服务器转发
		peerClientId, _ := cm.GetPeerInfo()
		if peerClientId != "" {
			// 协议格式: [魔数4B] + [0x02] + [targetId长度(1B)] + [targetId] + [数据长度(2B)] + [TUN数据]
			targetIdBytes := []byte(peerClientId)
			dataLen := len(tunData)
			packet := make([]byte, 4+1+1+len(targetIdBytes)+2+dataLen)

			// 魔数和模式标识
			packet[0] = 0x12
			packet[1] = 0x34
			packet[2] = 0x56
			packet[3] = 0x78
			packet[4] = 0x02 // 中转模式标识

			// targetId长度和targetId
			packet[5] = byte(len(targetIdBytes))
			copy(packet[6:6+len(targetIdBytes)], targetIdBytes)

			// 数据长度
			packet[6+len(targetIdBytes)] = byte(dataLen >> 8)
			packet[6+len(targetIdBytes)+1] = byte(dataLen & 0xFF)

			// TUN数据
			copy(packet[6+len(targetIdBytes)+2:], tunData)

			// 直接发送二进制数据到服务器
			_, err := conn.WriteToUDP(packet, serverAddr)
			if err != nil {
				glog.Errorf("[TUN]中转模式：发送数据失败：%v", err)
			}
		} else {
			glog.Warning("[TUN]中转模式：无法向peer发送包,peerClientId为空")
		}
	} else {
		glog.Warning("[TUN]未连接状态：无法发送数据包")
	}
}
