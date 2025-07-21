// Package player provides a heartbeat manager for managing player connections and sending heartbeat signals.
// It ensures that players are still connected and can perform periodic tasks.
package player

import (
	"sync"
	"time"
)

// Default heartbeat interval
var DefaultHeartbeatInterval = 5

type Heartbeat struct {
	players map[*Player]struct{}
	mu      sync.Mutex
	ticker  *time.Ticker
	stop    chan struct{}
}

// GlobalHeartbeat is a singleton instance of Heartbeat
// It is used to manage the heartbeat of all players in the system.
var GlobalHeartbeat *Heartbeat = NewHeartbeat()

func NewHeartbeat() *Heartbeat {
	return &Heartbeat{
		players: make(map[*Player]struct{}),
		stop:    make(chan struct{}),
	}
}

// Heartbeat starts the heartbeat process for all registered players at the specified interval.
// It sends a heartbeat signal to each player at the specified interval.
func (hm *Heartbeat) Start(interval time.Duration) {
	hm.ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-hm.ticker.C:
				hm.mu.Lock()
				for player := range hm.players {
					player.Heartbeat()
				}
				hm.mu.Unlock()
			case <-hm.stop:
				return
			}
		}
	}()
}

// Stop stops the heartbeat process and closes the stop channel.
// It should be called when the application is shutting down to clean up resources.
func (hm *Heartbeat) Stop() {
	hm.ticker.Stop()
	close(hm.stop)
}

// Register adds a player to the heartbeat manager.
// It should be called when a player is created or connected to the server.
func (hm *Heartbeat) Register(player *Player) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.players[player] = struct{}{}
}

// Unregister removes a player from the heartbeat manager.
// It should be called when a player is disconnected or removed from the server.
func (hm *Heartbeat) Unregister(player *Player) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.Stop()

	delete(hm.players, player)
}
