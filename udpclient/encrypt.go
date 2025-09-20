package main

import (
	"crypto/md5"
	"encoding/hex"
)

// HashMD5 计算字符串的 MD5 哈希值，返回 32 位十六进制字符串
func HashMD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
