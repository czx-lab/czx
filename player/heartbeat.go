// Package player provides a heartbeat manager for managing player connections and sending heartbeat signals.
// It ensures that players are still connected and can perform periodic tasks.
package player

import (
	"sync/atomic"
	"time"

	"github.com/czx-lab/czx/container/cmap"
	"github.com/czx-lab/czx/container/recycler"
)

// Default heartbeat interval
var DefaultHeartbeatInterval = 5

type (
	HeartbeatConf struct {
		cmap.Option[*Player]
	}
	// Heartbeat manages the heartbeat process for players.
	Heartbeat struct {
		players *cmap.Shareded[*Player, struct{}]
		ticker  *time.Ticker
		stop    chan struct{}
		closed  atomic.Bool
	}
)

// GlobalHeartbeat is a singleton instance of Heartbeat
// It is used to manage the heartbeat of all players in the system.
var GlobalHeartbeat *Heartbeat = NewHeartbeat(HeartbeatConf{
	Option: cmap.Option[*Player]{
		Count: 10,
	},
}, nil)

func NewHeartbeat(conf HeartbeatConf, r recycler.Recycler) *Heartbeat {
	return &Heartbeat{
		players: cmap.NewSharded[*Player, struct{}](
			conf.Option, r,
		),
		stop: make(chan struct{}),
	}
}

// Heartbeat starts the heartbeat process for all registered players at the specified interval.
// It sends a heartbeat signal to each player at the specified interval.
func (hm *Heartbeat) Start(interval time.Duration) {
	if !hm.closed.CompareAndSwap(false, true) {
		return
	}

	hm.stop = make(chan struct{})
	hm.ticker = time.NewTicker(interval)
	go func() {
		defer func() {
			hm.ticker.Stop()
			hm.ticker = nil
		}()

		for {
			select {
			case <-hm.ticker.C:
				hm.players.Iterator(func(s1 *Player, s2 struct{}) bool {
					s1.Heartbeat()
					return true
				})
			case <-hm.stop:
				return
			}
		}
	}()
}

// Stop stops the heartbeat process and closes the stop channel.
// It should be called when the application is shutting down to clean up resources.
func (hm *Heartbeat) Stop() {
	if !hm.closed.CompareAndSwap(true, false) {
		return
	}

	select {
	case <-hm.stop:
	default:
		close(hm.stop)
	}

	hm.players.Clear()
}

// Register adds a player to the heartbeat manager.
// It should be called when a player is created or connected to the server.
func (hm *Heartbeat) Register(player *Player) {
	hm.players.Set(player, struct{}{})
}

// Unregister removes a player from the heartbeat manager.
// It should be called when a player is disconnected or removed from the server.
func (hm *Heartbeat) Unregister(player *Player) {
	hm.players.Delete(player)
}
