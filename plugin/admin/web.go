package admin

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/DrmagicE/gmqtt/server"
)

//go:embed web/*
var dashboardAssets embed.FS

type dashboardQoS struct {
	Level    uint8  `json:"level"`
	Received string `json:"received"`
	Sent     string `json:"sent"`
	Dropped  string `json:"dropped"`
}

type dashboardOverview struct {
	ActiveSessions       string         `json:"active_sessions"`
	InactiveSessions     string         `json:"inactive_sessions"`
	ConnectionsTotal     string         `json:"connections_total"`
	DisconnectionsTotal  string         `json:"disconnections_total"`
	SubscriptionsCurrent string         `json:"subscriptions_current"`
	SubscriptionsTotal   string         `json:"subscriptions_total"`
	MessagesReceived     string         `json:"messages_received"`
	MessagesSent         string         `json:"messages_sent"`
	MessagesDropped      string         `json:"messages_dropped"`
	InflightCurrent      string         `json:"inflight_current"`
	QueuedCurrent        string         `json:"queued_current"`
	PacketsReceived      string         `json:"packets_received"`
	PacketsSent          string         `json:"packets_sent"`
	BytesReceived        string         `json:"bytes_received"`
	BytesSent            string         `json:"bytes_sent"`
	QoS                  []dashboardQoS `json:"qos"`
}

func counter(v uint64) string {
	return strconv.FormatUint(v, 10)
}

func newDashboardHandler() (http.Handler, error) {
	assets, err := fs.Sub(dashboardAssets, "web")
	if err != nil {
		return nil, err
	}
	files := http.FileServer(http.FS(assets))
	return http.StripPrefix("/admin/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		files.ServeHTTP(w, r)
	})), nil
}

type httpRouteRegistrar interface {
	RegisterHTTPRoute(pattern string, handler http.Handler)
}

func (a *Admin) registerDashboardHTTP(g server.APIRegistrar) error {
	routes, ok := g.(httpRouteRegistrar)
	if !ok {
		return nil
	}
	handler, err := newDashboardHandler()
	if err != nil {
		return err
	}
	routes.RegisterHTTPRoute("/admin/", handler)
	routes.RegisterHTTPRoute("/admin/api/overview", http.HandlerFunc(a.handleDashboardOverview))
	return nil
}

func (a *Admin) handleDashboardOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if a.statsReader == nil {
		http.Error(w, "statistics unavailable", http.StatusServiceUnavailable)
		return
	}

	stats := a.statsReader.GetGlobalStats()
	qos := []dashboardQoS{
		{
			Level:    0,
			Received: counter(stats.MessageStats.Qos0.ReceivedTotal),
			Sent:     counter(stats.MessageStats.Qos0.SentTotal),
			Dropped:  counter(stats.MessageStats.Qos0.GetDroppedTotal()),
		},
		{
			Level:    1,
			Received: counter(stats.MessageStats.Qos1.ReceivedTotal),
			Sent:     counter(stats.MessageStats.Qos1.SentTotal),
			Dropped:  counter(stats.MessageStats.Qos1.GetDroppedTotal()),
		},
		{
			Level:    2,
			Received: counter(stats.MessageStats.Qos2.ReceivedTotal),
			Sent:     counter(stats.MessageStats.Qos2.SentTotal),
			Dropped:  counter(stats.MessageStats.Qos2.GetDroppedTotal()),
		},
	}

	response := dashboardOverview{
		ActiveSessions:       counter(stats.ConnectionStats.ActiveCurrent),
		InactiveSessions:     counter(stats.ConnectionStats.InactiveCurrent),
		ConnectionsTotal:     counter(stats.ConnectionStats.ConnectedTotal),
		DisconnectionsTotal:  counter(stats.ConnectionStats.DisconnectedTotal),
		SubscriptionsCurrent: counter(stats.SubscriptionStats.SubscriptionsCurrent),
		SubscriptionsTotal:   counter(stats.SubscriptionStats.SubscriptionsTotal),
		MessagesReceived: counter(stats.MessageStats.Qos0.ReceivedTotal +
			stats.MessageStats.Qos1.ReceivedTotal + stats.MessageStats.Qos2.ReceivedTotal),
		MessagesSent: counter(stats.MessageStats.Qos0.SentTotal +
			stats.MessageStats.Qos1.SentTotal + stats.MessageStats.Qos2.SentTotal),
		MessagesDropped: counter(stats.MessageStats.GetDroppedTotal()),
		InflightCurrent: counter(stats.MessageStats.InflightCurrent),
		QueuedCurrent:   counter(stats.MessageStats.QueuedCurrent),
		PacketsReceived: counter(stats.PacketStats.ReceivedTotal.Total),
		PacketsSent:     counter(stats.PacketStats.SentTotal.Total),
		BytesReceived:   counter(stats.PacketStats.BytesReceived.Total),
		BytesSent:       counter(stats.PacketStats.BytesSent.Total),
		QoS:             qos,
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Warn("write dashboard overview failed")
	}
}
