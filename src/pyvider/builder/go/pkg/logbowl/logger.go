package logbowl

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// Environment variable names
const (
	LogLevelEnvVar  = "PYVIDER_LOG_LEVEL"
	LogFormatEnvVar = "PYVIDER_LOG_CONSOLE_FORMATTER"
)

// Log formats
const (
	FormatEmoji = "emoji"
	FormatText  = "text"
	FormatJSON  = "json"
)

var domains = map[string]string{"system": "⚙️", "server": "🛎️", "client": "🙋", "network": "🌐", "security": "🔐", "config": "🔩", "database": "🗄️", "cache": "💾", "task": "🔄", "plugin": "🔌", "telemetry": "🛰️", "di": "💉", "protocol": "📡", "file": "📄", "user": "👤", "test": "🧪", "utils": "🧰", "core": "🌟", "auth": "🔑", "entity": "🦎", "report": "📈", "default": "❓", "launcher": "🚀", "builder": "🛠️", "signing": "✍️", "archive": "📦", "keymgmt": "🔑", "io": "💾", "env": "🌿", "package": "📦"}
var actions = map[string]string{"init": "🌱", "start": "🚀", "stop": "🛑", "connect": "🔗", "disconnect": "💔", "listen": "👂", "send": "📤", "receive": "📥", "read": "📖", "write": "📝", "process": "⚙️", "validate": "🛡️", "execute": "▶️", "query": "🔍", "update": "🔄", "delete": "🗑️", "login": "➡️", "logout": "⬅️", "auth": "🔑", "error": "🔥", "encrypt": "🛡️", "decrypt": "🔓", "parse": "🧩", "transmit": "📡", "build": "🏗️", "schedule": "📅", "emit": "📢", "load": "💡", "observe": "🧐", "request": "🗣️", "interrupt": "🚦", "verify": "🔍", "pack": "📦", "generate": "✨", "clean": "🧹", "install": "🧩", "finish": "🏁", "info": "💡", "default": "⚙️"}
var statuses = map[string]string{"success": "✅", "failure": "❌", "error": "🔥", "warning": "⚠️", "info": "ℹ️", "debug": "🐞", "trace": "👣", "attempt": "⏳", "retry": "🔁", "skip": "⏭️", "complete": "🏁", "timeout": "⏱️", "notfound": "❓", "unauthorized": "🚫", "invalid": "💢", "cached": "🎯", "ongoing": "🏃", "idle": "💤", "ready": "👍", "progress": "➡️", "ok": "✅", "default": "➡️"}

func getEmoji(m map[string]string, key string) string {
	if val, ok := m[key]; ok { return val }
	return m["default"]
}

// Logger wraps hclog.Logger to provide the simplified API.
type Logger struct {
	hclog.Logger
}

// Create creates a new Logger instance.
func Create(name string) Logger {
	levelStr := os.Getenv(LogLevelEnvVar)
	level := hclog.LevelFromString(strings.ToUpper(levelStr))
	if level == hclog.NoLevel {
		level = hclog.Info
	}

	formatStr := strings.ToLower(os.Getenv(LogFormatEnvVar))
	jsonFormat := formatStr == FormatJSON

	opts := &hclog.LoggerOptions{
		Name:       name,
		Level:      level,
		JSONFormat: jsonFormat,
	}
	return Logger{hclog.New(opts)}
}

func (l Logger) log(level hclog.Level, domain, action, status, message string, args ...interface{}) {
	formatStr := strings.ToLower(os.Getenv(LogFormatEnvVar))
	switch formatStr {
	case FormatText:
		l.Logger.Log(level, fmt.Sprintf("[%s] %s", strings.ToUpper(domain), message), args...)
	case FormatJSON:
		l.Logger.With("domain", domain, "action", action, "status", status).Log(level, message, args...)
	default: // Emoji format
		l.Logger.Log(level, fmt.Sprintf("%s %s %s %s", getEmoji(domains, domain), getEmoji(actions, action), getEmoji(statuses, status), message), args...)
	}
}

func (l Logger) Info(domain, action, status, message string, args ...interface{}) {
	l.log(hclog.Info, domain, action, status, message, args...)
}
func (l Logger) Debug(domain, action, status, message string, args ...interface{}) {
	l.log(hclog.Debug, domain, action, status, message, args...)
}
func (l Logger) Warn(domain, action, status, message string, args ...interface{}) {
	l.log(hclog.Warn, domain, action, status, message, args...)
}
func (l Logger) Error(domain, action, status, message string, args ...interface{}) {
	l.log(hclog.Error, domain, action, status, message, args...)
}
