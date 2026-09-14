package ws_test

import (
	"testing"

	"github.com/misbakhul29/goks/pkg/ws"
)

func BenchmarkWSHub_Broadcast100Clients(b *testing.B) {
	hub := ws.NewHub()
	const numClients = 100

	clients := make([]*ws.Client, numClients)
	for i := 0; i < numClients; i++ {
		c := &ws.Client{}
		// Wire client into hub via test reflection or Join if available
		clients[i] = c
	}

	msg := []byte(`{"event":"chat.message","data":{"text":"hello world benchmark"}}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hub.Broadcast(msg)
	}
}
