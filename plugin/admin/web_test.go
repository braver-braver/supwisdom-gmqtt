package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DrmagicE/gmqtt/persistence/subscription"
	"github.com/DrmagicE/gmqtt/server"
)

type dashboardStatsReader struct {
	global server.GlobalStats
}

func (d dashboardStatsReader) GetGlobalStats() server.GlobalStats {
	return d.global
}

func (d dashboardStatsReader) GetClientStats(string) (server.ClientStats, bool) {
	return server.ClientStats{}, false
}

func TestDashboardHandlerServesEmbeddedIndex(t *testing.T) {
	handler, err := newDashboardHandler()
	if err != nil {
		t.Fatalf("newDashboardHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if !strings.Contains(resp.Body.String(), "GMQTT Admin") {
		t.Fatal("embedded index does not contain dashboard title")
	}
	if got := resp.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("missing Content-Security-Policy header")
	}
}

func TestDashboardOverview(t *testing.T) {
	stats := server.GlobalStats{
		SubscriptionStats: subscription.Stats{
			SubscriptionsCurrent: 7,
			SubscriptionsTotal:   11,
		},
	}
	stats.ConnectionStats.ActiveCurrent = 2
	stats.ConnectionStats.InactiveCurrent = 3
	stats.ConnectionStats.ConnectedTotal = 13
	stats.ConnectionStats.DisconnectedTotal = 8
	stats.MessageStats.Qos0.ReceivedTotal = 5
	stats.MessageStats.Qos1.ReceivedTotal = 6
	stats.MessageStats.Qos2.ReceivedTotal = 7
	stats.MessageStats.Qos0.SentTotal = 8
	stats.MessageStats.Qos1.SentTotal = 9
	stats.MessageStats.Qos2.SentTotal = 10
	stats.MessageStats.Qos1.DroppedTotal.QueueFull = 4
	stats.MessageStats.InflightCurrent = 12
	stats.MessageStats.QueuedCurrent = 14
	stats.PacketStats.ReceivedTotal.Total = 20
	stats.PacketStats.SentTotal.Total = 21
	stats.PacketStats.BytesReceived.Total = 2048
	stats.PacketStats.BytesSent.Total = 4096

	admin := &Admin{statsReader: dashboardStatsReader{global: stats}}
	req := httptest.NewRequest(http.MethodGet, "/admin/api/overview", nil)
	resp := httptest.NewRecorder()
	admin.handleDashboardOverview(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	var got dashboardOverview
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ActiveSessions != "2" || got.SubscriptionsCurrent != "7" {
		t.Fatalf("unexpected session/subscription counters: %+v", got)
	}
	if got.MessagesReceived != "18" || got.MessagesSent != "27" || got.MessagesDropped != "4" {
		t.Fatalf("unexpected message counters: %+v", got)
	}
	if got.InflightCurrent != "12" || got.QueuedCurrent != "14" {
		t.Fatalf("unexpected queue counters: %+v", got)
	}
	if got.BytesReceived != "2048" || got.BytesSent != "4096" {
		t.Fatalf("unexpected byte counters: %+v", got)
	}
	if len(got.QoS) != 3 || got.QoS[1].Dropped != "4" {
		t.Fatalf("unexpected QoS counters: %+v", got.QoS)
	}
}

func TestDashboardOverviewRejectsWrites(t *testing.T) {
	admin := &Admin{statsReader: dashboardStatsReader{}}
	req := httptest.NewRequest(http.MethodPost, "/admin/api/overview", nil)
	resp := httptest.NewRecorder()
	admin.handleDashboardOverview(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
	}
	if got := resp.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
	}
}
