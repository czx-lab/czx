package main

import (
	"czx/agent"
	"czx/eventbus"
	"czx/network"
	"czx/network/ws"
	"fmt"
	"time"
)

func main() {
	gateConf := &agent.GateConf{
		WsServerConf: ws.WsServerConf{
			Addr:            ":8080",
			MaxConn:         100000,
			PendingWriteNum: 1000,
			MaxMsgSize:      4096,
			Timeout:         10 * time.Second,
		},
	}

	closeFlag := make(chan struct{})
	gateway := agent.NewGate(*gateConf).WithPreConn(func(a network.Agent, phm network.ClientAddrMessage) {
		// Pre-handler message logic here
		fmt.Println("Pre-handler message:", phm)
	}).WithFlag(closeFlag)

	go func() {
		eventbus.DefaultBus.Subscribe(eventbus.EvtNewAgent, func(message any) {
			fmt.Println(message.(network.Agent).LocalAddr().String())
		})
	}()

	fmt.Printf("Starting websocket server at %s...\n", gateConf.WsServerConf.Addr)
	gateway.Start()
}
