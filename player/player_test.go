package player

import (
	"czx/network"
	"testing"
	"time"
)

func TestPlayer(t *testing.T) {
	t.Run("TestNewPlayer", func(t *testing.T) {
		p := NewPlayer(nil)
		p.WithData("player data")
		p.WithID("player_id_1")
		p.SetHeartbeatLogic(func(agent network.Agent) {
			// Placeholder for heartbeat logic
			t.Log("Heartbeat logic executed")
		})

		t.Log(p.Data())
		t.Log(p.ID())
		t.Log(p.Agent())
	})

	t.Run("TestManger", func(t *testing.T) {
		p := NewPlayer(nil)
		p.WithData("player data")
		p.WithID("player_id_1")
		p.SetHeartbeatLogic(func(agent network.Agent) {
			// Placeholder for heartbeat logic
			t.Log("Heartbeat logic executed")
		})

		t.Log(p.Data())
		t.Log(p.ID())
		t.Log(p.Agent())

		m := NewPlayerManager(&ManagerConf{})
		m.Add(p)
		t.Log(m.Player("player_id_1"))
		m.Start()

		time.Sleep(time.Second * 25)
		m.Remove("player_id_1", false)

		m.Add(p)
		t.Log("player_id_1 added again")

		time.Sleep(time.Second * 15)
		m.Close()

		t.Log("manager closed")
	})
}
