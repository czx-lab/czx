package player

import (
	"github.com/czx-lab/czx/network"
)

type Player struct {
	id string
	// Placeholder for player data, can be any type
	data           any
	agent          network.Agent
	heartbeatLogic func(network.Agent)
}

func NewPlayer(agent network.Agent) *Player {
	return &Player{
		agent: agent,
	}
}

func (p *Player) ID() string {
	return p.id
}

// WithID sets the ID for the player. This can be used to identify the player in the system.
func (p *Player) WithID(id string) {
	p.id = id
}

// Agent retrieves the agent associated with the player. This can be used to send messages to the player or receive messages from the player.
// For example, it can be used to send game updates, notifications, etc.
func (p *Player) Agent() network.Agent {
	return p.agent
}

// WithAgent sets the agent for the player. This can be used to associate a network connection with the player.
// For example, it can be used to send messages to the player or receive messages from the player.
func (p *Player) WithAgent(agent network.Agent) {
	p.agent = agent
}

// Data retrieves the data associated with the player. This can be any type of data that is relevant to the player.
// For example, it can be used to store player statistics, preferences, etc.
func (p *Player) Data() any {
	return p.data
}

// WithData sets the data for the player. This can be used to store any additional information related to the player.
// For example, it can be used to store player statistics, preferences, etc.
func (p *Player) WithData(data any) {
	p.data = data
}

// Heartbeat sends a heartbeat signal to the player agent.
// It can be used to check if the player is still connected or to perform any periodic task.
func (p *Player) SetHeartbeatLogic(logic func(network.Agent)) {
	p.heartbeatLogic = logic
}

// Send a heartbeat signal to the player agent
func (p *Player) Heartbeat() {
	if p.heartbeatLogic == nil {
		return
	}

	p.heartbeatLogic(p.agent)
}

// StopHeartbeat stops sending heartbeat signals to the player agent
// This is typically called when the player is no longer needed or when the game session ends.
func (p *Player) StopHeartbeat() {
	// Unregister from heartbeat manager
	GlobalHeartbeat.Unregister(p)
}

// Close the player connection and clean up resources
func (p *Player) Close() {
	if p.agent != nil {
		p.agent.Close()
	}

	// Unregister from heartbeat manager
	GlobalHeartbeat.Unregister(p)
}

// Destroy cleans up the player resources and unregisters from the heartbeat manager
// This is typically called when the player is no longer needed or when the game session ends.
func (p *Player) Destroy() {
	// Unregister from heartbeat manager
	GlobalHeartbeat.Unregister(p)

	if p.agent != nil {
		p.agent.Destroy()
	}
}
