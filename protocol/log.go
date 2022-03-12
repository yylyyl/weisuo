package protocol

import (
	"fmt"
	"runtime"
	"weisuo/logger"
)

func (req *request) logDebugf(format string, a ...interface{}) {
	if req.h.Logger != nil && req.h.LogLevel >= logger.LogLevelDebug {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "unknown"
			line = 0
		}
		req.h.Logger.Debug(fmt.Sprintf("S %s %s:%d %s ", req.id.String(), file, line, req.realIp) + fmt.Sprintf(format, a...))
	}
}
func (req *request) logInfof(format string, a ...interface{}) {
	if req.h.Logger != nil && req.h.LogLevel >= logger.LogLevelInfo {
		req.h.Logger.Info(fmt.Sprintf("S %s %s ", req.id.String(), req.realIp) + fmt.Sprintf(format, a...))
	}
}
func (req *request) logWarnf(format string, a ...interface{}) {
	if req.h.Logger != nil && req.h.LogLevel >= logger.LogLevelWarn {
		req.h.Logger.Warn(fmt.Sprintf("S %s %s ", req.id.String(), req.realIp) + fmt.Sprintf(format, a...))
	}
}
func (req *request) logErrorf(format string, a ...interface{}) {
	if req.h.Logger != nil && req.h.LogLevel >= logger.LogLevelError {
		req.h.Logger.Error(fmt.Sprintf("S %s %s ", req.id.String(), req.realIp) + fmt.Sprintf(format, a...))
	}
}

func (c *connTcp) logDebugf(format string, a ...interface{}) {
	if c.logger != nil && c.logLevel >= logger.LogLevelDebug {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "unknown"
			line = 0
		}
		c.logger.Debug(fmt.Sprintf("%s %s:%d ", c.id.String(), file, line) + fmt.Sprintf(format, a...))
	}
}
func (c *connTcp) logInfof(format string, a ...interface{}) {
	if c.logger != nil && c.logLevel >= logger.LogLevelInfo {
		c.logger.Info(fmt.Sprintf("%s ", c.id.String()) + fmt.Sprintf(format, a...))
	}
}
func (c *connTcp) logWarnf(format string, a ...interface{}) {
	if c.logger != nil && c.logLevel >= logger.LogLevelWarn {
		c.logger.Warn(fmt.Sprintf("%s ", c.id.String()) + fmt.Sprintf(format, a...))
	}
}
func (c *connTcp) logErrorf(format string, a ...interface{}) {
	if c.logger != nil && c.logLevel >= logger.LogLevelError {
		c.logger.Error(fmt.Sprintf("%s ", c.id.String()) + fmt.Sprintf(format, a...))
	}
}
