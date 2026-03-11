package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	var tt = []struct {
		caseName string
		fileName string
		hasErr   bool
		expected Config
	}{
		{
			caseName: "defaultConfig",
			fileName: "",
			hasErr:   false,
			expected: DefaultConfig(),
		},
	}

	for _, v := range tt {
		t.Run(v.caseName, func(t *testing.T) {
			a := assert.New(t)
			c, err := ParseConfig(v.fileName)
			if v.hasErr {
				a.NotNil(err)
			} else {
				a.Nil(err)
			}
			a.Equal(v.expected, c)
		})
	}
}

func TestLogConfigValidate(t *testing.T) {
	require.NoError(t, DefaultConfig().Log.Validate())
	require.Equal(t, "error", DefaultConfig().Log.StacktraceLevel)
	require.True(t, DefaultConfig().Log.EnableCaller)

	err := LogConfig{
		Level:           "info",
		Format:          "text",
		EnableCaller:    true,
		StacktraceLevel: "fatal",
	}.Validate()
	require.EqualError(t, err, "invalid stacktrace level: fatal")
}
