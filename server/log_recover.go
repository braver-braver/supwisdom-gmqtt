package server

import (
	"runtime/debug"

	"go.uber.org/zap"
)

// LogPanic recovers panic and writes stacktrace for preserving crash scene.
func LogPanic(log *zap.Logger, where string) {
	r := recover()
	if r == nil {
		return
	}
	if log == nil {
		log = zaplog
	}
	log.Error("panic recovered",
		zap.String("where", where),
		zap.Any("panic", r),
		zap.ByteString("stack", debug.Stack()),
	)
}
