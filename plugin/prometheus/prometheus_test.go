package prometheus

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeDashboard(t *testing.T) {
	p := &Prometheus{
		enableDashboard: true,
	}

	// Create a test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Call the dashboard handler
	p.serveDashboard(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check content type
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

	// Check that response contains HTML
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "<!DOCTYPE html>")
	assert.Contains(t, string(body), "Gmqtt Metrics Dashboard")
}

func TestConfigDefaults(t *testing.T) {
	assert.Equal(t, ":8082", DefaultConfig.ListenAddress)
	assert.Equal(t, "/metrics", DefaultConfig.Path)
	assert.True(t, DefaultConfig.EnableDashboard)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ListenAddress:   ":8082",
				Path:            "/metrics",
				EnableDashboard: true,
			},
			wantErr: false,
		},
		{
			name: "invalid listen address",
			config: Config{
				ListenAddress:   "invalid",
				Path:            "/metrics",
				EnableDashboard: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


