package server

import (
	"testing"

	"github.com/DrmagicE/gmqtt/persistence/subscription/mem"
	"github.com/DrmagicE/gmqtt/pkg/packets"
)

func TestStatsManagerMessageQoSAccounting(t *testing.T) {
	const clientID = "client"
	tests := []struct {
		name string
		qos  uint8
	}{
		{name: "qos0", qos: packets.Qos0},
		{name: "qos1", qos: packets.Qos1},
		{name: "qos2", qos: packets.Qos2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := newStatsManager(mem.NewStore())
			stats.messageReceived(tt.qos, clientID)
			stats.messageSent(tt.qos, clientID)

			clientStats, ok := stats.GetClientStats(clientID)
			if !ok {
				t.Fatal("client stats not found")
			}

			want := [3]uint64{}
			want[tt.qos] = 1

			gotReceived := [3]uint64{
				clientStats.MessageStats.Qos0.ReceivedTotal,
				clientStats.MessageStats.Qos1.ReceivedTotal,
				clientStats.MessageStats.Qos2.ReceivedTotal,
			}
			if gotReceived != want {
				t.Fatalf("received totals = %v, want %v", gotReceived, want)
			}

			gotSent := [3]uint64{
				clientStats.MessageStats.Qos0.SentTotal,
				clientStats.MessageStats.Qos1.SentTotal,
				clientStats.MessageStats.Qos2.SentTotal,
			}
			if gotSent != want {
				t.Fatalf("sent totals = %v, want %v", gotSent, want)
			}
		})
	}
}

func TestStatsManagerInflightAccounting(t *testing.T) {
	const clientID = "client"
	stats := newStatsManager(mem.NewStore())

	stats.addInflight(clientID, 3)

	clientStats, ok := stats.GetClientStats(clientID)
	if !ok {
		t.Fatal("client stats not found")
	}
	if got := clientStats.MessageStats.InflightCurrent; got != 3 {
		t.Fatalf("client inflight = %d, want 3", got)
	}
	if got := stats.GetGlobalStats().MessageStats.InflightCurrent; got != 3 {
		t.Fatalf("global inflight = %d, want 3", got)
	}

	stats.decInflight(clientID, 2)

	clientStats, ok = stats.GetClientStats(clientID)
	if !ok {
		t.Fatal("client stats not found after decrement")
	}
	if got := clientStats.MessageStats.InflightCurrent; got != 1 {
		t.Fatalf("client inflight after decrement = %d, want 1", got)
	}
	if got := stats.GetGlobalStats().MessageStats.InflightCurrent; got != 1 {
		t.Fatalf("global inflight after decrement = %d, want 1", got)
	}
}

func TestPacketBytesCopyIncludesAuth(t *testing.T) {
	want := PacketBytes{
		Auth:    3,
		Connect: 2,
		Total:   5,
	}

	if got := want.copy(); got != want {
		t.Fatalf("copy() = %+v, want %+v", got, want)
	}
}
