package player

import (
	"errors"
	"slices"
	"sync"
	"time"
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
		players map[string]*Player
	}
	// BroadcastMessage is a struct that represents a message to be broadcasted to players.
	BroadcastMessage struct {
		Code uint16
		Data any
	}
)

func NewPlayerManager(conf *ManagerConf) *PlayerManager {
	return &PlayerManager{
		conf:    conf,
		players: make(map[string]*Player),
	}
}

// Add adds a new player to the player manager. If the player already exists, it returns an error.
// It returns an error if the player already exists.
func (p *PlayerManager) Add(player *Player) error {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.players[player.ID()]; ok {
		return ErrPlayerAdded
	}

	p.players[player.ID()] = player

	// Register the player with the heartbeat manager
	GlobalHeartbeat.Register(player)
	return nil
}

// Player retrieves the agent from the player. If the agent is not found, it returns an error.
// This function is not thread-safe, so it should be called with the player locked.
func (p *PlayerManager) Player(id string) (*Player, error) {
	p.RLock()
	defer p.RUnlock()

	player, ok := p.players[id]
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

	players := make([]*Player, 0, len(p.players))
	for _, player := range p.players {
		players = append(players, player)
	}
	return players
}

// Num returns the number of players in the player manager.
func (p *PlayerManager) Num() int {
	p.RLock()
	defer p.RUnlock()

	return len(p.players)
}

// Remove removes a player from the player manager. If the player does not exist, it does nothing.
func (p *PlayerManager) Remove(id string, destroy bool) error {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.players[id]; !ok {
		return ErrPlayerNotFound
	}

	// Unregister from heartbeat manager
	if destroy {
		p.players[id].Destroy()
	} else {
		p.players[id].Close()
	}

	delete(p.players, id)
	return nil
}

// Rang iterates over all players and applies the provided function to each player.
func (p *PlayerManager) Rang(fn func(*Player)) error {
	p.RLock()
	defer p.RUnlock()

	for _, player := range p.players {
		fn(player)
	}
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

	for _, player := range p.players {
		player.Close()
	}
	p.players = make(map[string]*Player)
}
