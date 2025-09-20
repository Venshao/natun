// Package gjson - 自定义 JSON 模块
package gjson

import (
	"encoding/json"
	"fmt"
)

// New 新建一个 JSON 对象
func New(data interface{}) *Json {
	return &Json{data}
}

// Json - 定义结构体封装原始数据
type Json struct {
	data interface{}
}

// ToJson - 序列化 JSON 数据
func (j *Json) ToJson() ([]byte, error) {
	return json.Marshal(j.data)
}

// GetString - 获取指定字段的字符串值
func (j *Json) GetString(key string) string {
	if obj, ok := j.data.(map[string]interface{}); ok {
		if val, exists := obj[key]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

// GetInt - 获取指定字段的整数值
func (j *Json) GetInt(key string) int {
	if obj, ok := j.data.(map[string]interface{}); ok {
		if val, exists := obj[key]; exists {
			if i, ok := val.(float64); ok { // json.Marshal 将数字解析为 float64
				return int(i)
			}
		}
	}
	return 0
}

// GetInt8 - 获取指定字段的 int8 值
func (j *Json) GetInt8(key string) int8 {
	if obj, ok := j.data.(map[string]interface{}); ok {
		if val, exists := obj[key]; exists {
			if i, ok := val.(float64); ok { // json.Marshal 将数字解析为 float64
				return int8(i)
			}
		}
	}
	return 0
}

// GetInt64 - 获取指定字段的 int64 值
func (j *Json) GetInt64(key string) int64 {
	if obj, ok := j.data.(map[string]interface{}); ok {
		if val, exists := obj[key]; exists {
			if i, ok := val.(float64); ok { // json.Marshal 将数字解析为 float64
				return int64(i)
			}
		}
	}
	return 0
}

// Set - 设置指定字段的值
func (j *Json) Set(key string, value interface{}) error {
	if obj, ok := j.data.(map[string]interface{}); ok {
		obj[key] = value
		return nil
	}
	return fmt.Errorf("无法设置值，数据格式不正确")
}

// LoadContent - 解析 JSON 字符串
func LoadContent(content string) (*Json, error) {
	var data interface{}
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}
	return &Json{data}, nil
}
