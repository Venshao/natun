// log.go - 自定义日志模块
package glog

import (
	"log"
	"strings"
)

// LogLevel 日志等级
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	currentLevel LogLevel = INFO
	levelNames            = map[LogLevel]string{
		DEBUG: "DEBU",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERRO",
		FATAL: "FATA",
	}
)

// SetLevel 设置日志等级
func SetLevel(level LogLevel) {
	currentLevel = level
}

// SetLevelString 通过字符串设置日志等级
func SetLevelString(levelStr string) {
	switch strings.ToUpper(levelStr) {
	case "DEBUG", "DEBU":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARN", "WARNING":
		currentLevel = WARN
	case "ERROR", "ERRO":
		currentLevel = ERROR
	case "FATAL", "FATA":
		currentLevel = FATAL
	default:
		currentLevel = INFO
	}
}

// shouldLog 检查是否应该输出指定等级的日志
func shouldLog(level LogLevel) bool {
	return level >= currentLevel
}

// logWithLevel 输出指定等级的日志
func logWithLevel(level LogLevel, format string, args ...interface{}) {
	if !shouldLog(level) {
		return
	}

	levelName := levelNames[level]
	if len(args) > 0 {
		log.Printf(levelName+": "+format, args...)
	} else {
		log.Printf(levelName + ": " + format)
	}
}

// Debugf - 输出调试日志
func Debugf(format string, args ...interface{}) {
	logWithLevel(DEBUG, format, args...)
}

// Debug - 输出调试日志
func Debug(format string) {
	logWithLevel(DEBUG, format)
}

// Warningf - 输出警告日志
func Warningf(format string, args ...interface{}) {
	logWithLevel(WARN, format, args...)
}

// Warning - 输出警告日志
func Warning(format string) {
	logWithLevel(WARN, format)
}

// Errorf - 输出错误日志
func Errorf(format string, args ...interface{}) {
	logWithLevel(ERROR, format, args...)
}

// Infof - 输出信息日志
func Infof(format string, args ...interface{}) {
	logWithLevel(INFO, format, args...)
}

// Fatalf - 输出致命错误日志
func Fatalf(format string, args ...interface{}) {
	logWithLevel(FATAL, format, args...)
}

// GetCurrentLevel 获取当前日志等级
func GetCurrentLevel() LogLevel {
	return currentLevel
}

// GetCurrentLevelString 获取当前日志等级字符串
func GetCurrentLevelString() string {
	return levelNames[currentLevel]
}
