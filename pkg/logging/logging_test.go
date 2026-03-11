package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestWithCaller(t *testing.T) {
	buffer := &strings.Builder{}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(buffer),
		zapcore.DebugLevel,
	)
	logger := zap.New(core, zap.AddCaller())

	expectedLine := callerLine() + 1
	logThroughHelper(logger)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(buffer.String()), &payload))
	caller, ok := payload["caller"].(string)
	require.True(t, ok)
	require.Contains(t, caller, "logging_test.go")
	require.Equal(t, expectedLine, parseCallerLine(t, caller))
}

func TestErr(t *testing.T) {
	root := errors.New("root cause")
	wrapped := fmt.Errorf("operation failed: %w", root)

	fields := Err(wrapped)
	require.Len(t, fields, 3)
	require.Equal(t, "error", fields[0].Key)
	require.Equal(t, "error_type", fields[1].Key)
	require.Equal(t, "error_chain", fields[2].Key)
	require.Equal(t, []string{"operation failed: root cause", "root cause"}, ErrorChain(wrapped))
}

func logThroughHelper(logger *zap.Logger) {
	WithCaller(logger).Error("wrapped")
}

func callerLine() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

func parseCallerLine(t *testing.T, caller string) int {
	t.Helper()
	parts := strings.Split(caller, ":")
	require.Len(t, parts, 2)
	line, err := strconv.Atoi(parts[1])
	require.NoError(t, err)
	return line
}
