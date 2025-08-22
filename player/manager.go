package player

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/czx-lab/czx/container/cmap"
	"github.com/czx-lab/czx/container/recycler"
)

var (
	ErrPlayerAdded    = errors.New("player already added")
	ErrPlayerNotFound = errors.New("player not found")
)

type (
	ManagerConf struct {
		// Heartbeat interval in seconds
		HeartbeatInterval int
	}
	PlayerManager struct {
		sync.RWMutex
		conf    *ManagerConf
		players *cmap.CMap[string, *Player]
	}
	// BroadcastMessage is a struct that represents a message to be broadcasted to players.
	BroadcastMessage struct {
		Code uint16
		Data any
	}
)

func NewPlayerManager(conf *ManagerConf, r recycler.Recycler) *PlayerManager {
	ps := cmap.New[string, *Player]()
	return &PlayerManager{
		conf:    conf,
		players: ps.WithRecycler(r),
	}
}

// Add adds a new player to the player manager. If the player already exists, it returns an error.
// It returns an error if the player already exists.
func (p *PlayerManager) Add(player *Player) error {
	p.Lock()
	defer p.Unlock()
	if p.players.Has(player.ID()) {
		return ErrPlayerAdded
	}

	p.players.Set(player.ID(), player)

	// Register the player with the heartbeat manager
	GlobalHeartbeat.Register(player)
	return nil
}

// Player retrieves the agent from the player. If the agent is not found, it returns an error.
// This function is not thread-safe, so it should be called with the player locked.
func (p *PlayerManager) Player(id string) (*Player, error) {
	p.RLock()
	defer p.RUnlock()

	player, ok := p.players.Get(id)
	if !ok {
		return nil, ErrPlayerNotFound
	}
	return player, nil
}

// Start starts the heartbeat process for all registered players at the specified interval.
// It sends a heartbeat signal to each player at the specified interval.
func (p *PlayerManager) Start() {
	if p.conf.HeartbeatInterval == 0 {
		p.conf.HeartbeatInterval = DefaultHeartbeatInterval
	}

	GlobalHeartbeat.Start(time.Duration(p.conf.HeartbeatInterval) * time.Second)
}

// Players retrieves all agents from the player. This function is not thread-safe, so it should be called with the player locked.
// It returns a slice of agents.
func (p *PlayerManager) Players() []*Player {
	p.RLock()
	defer p.RUnlock()

	players := make([]*Player, 0, p.players.Len())
	p.players.Iterator(func(id string, player *Player) bool {
		players = append(players, player)
		return true
	})
	return players
}

// Num returns the number of players in the player manager.
func (p *PlayerManager) Num() int {
	p.RLock()
	defer p.RUnlock()

	return p.players.Len()
}

// Delete removes a player from the player manager by ID. If the player does not exist, it returns an error.
func (p *PlayerManager) Delete(id string) error {
	p.Lock()
	defer p.Unlock()

	if !p.players.Has(id) {
		return ErrPlayerNotFound
	}

	p.players.Delete(id)
	return nil
}

// Remove removes a player from the player manager. If the player does not exist, it does nothing.
func (p *PlayerManager) Remove(id string, destroy bool) error {
	p.Lock()
	defer p.Unlock()

	if !p.players.Has(id) {
		return ErrPlayerNotFound
	}

	// Unregister from heartbeat manager
	player, _ := p.players.Get(id)
	if destroy {
		player.Destroy()
	} else {
		player.Close()
	}

	p.players.Delete(id)
	return nil
}

// Rang iterates over all players and applies the provided function to each player.
func (p *PlayerManager) Rang(fn func(*Player)) error {
	p.RLock()
	defer p.RUnlock()

	p.players.Iterator(func(_ string, player *Player) bool {
		fn(player)
		return true
	})
	return nil
}

// Broadcast sends a message to all players.
// It can be used to send game updates, notifications, etc.
func (p *PlayerManager) Broadcast(msg BroadcastMessage) error {
	return p.Rang(func(player *Player) {
		if msg.Code == 0 {
			player.Agent().Write(msg.Data)
			return
		}

		player.Agent().WriteWithCode(msg.Code, msg.Data)
	})
}

// BroadcastExcepts sends a message to all players except the specified ones.
func (p *PlayerManager) BroadcastExcepts(msg BroadcastMessage, ids ...string) error {
	return p.Rang(func(player *Player) {
		if slices.Contains(ids, player.ID()) {
			return
		}

		if msg.Code == 0 {
			player.Agent().Write(msg.Data)
			return
		}

		player.Agent().WriteWithCode(msg.Code, msg.Data)
	})
}

// BroadcastByIds sends a message to players with the specified IDs.
// It can be used to send messages to specific players based on their IDs.
func (p *PlayerManager) BroadcastByIds(msg BroadcastMessage, ids ...string) error {
	return p.Rang(func(player *Player) {
		if !slices.Contains(ids, player.ID()) {
			return
		}

		if msg.Code == 0 {
			player.Agent().Write(msg.Data)
			return
		}

		player.Agent().WriteWithCode(msg.Code, msg.Data)
	})
}

// BroadcastByFunc sends a message to players that match the provided function.
// It can be used to send messages to specific players based on custom logic.
func (p *PlayerManager) BroadcastByFunc(msg BroadcastMessage, fn func(*Player) bool) error {
	return p.Rang(func(player *Player) {
		if !fn(player) {
			return
		}

		if msg.Code == 0 {
			player.Agent().Write(msg.Data)
			return
		}

		player.Agent().WriteWithCode(msg.Code, msg.Data)
	})
}

// Close closes all player connections and cleans up resources.
func (p *PlayerManager) Close() {
	p.RLock()
	defer p.RUnlock()

	p.players.Iterator(func(_ string, player *Player) bool {
		player.Close()
		return true
	})
	p.players.Clear()
}
