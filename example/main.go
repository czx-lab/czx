package main

import (
	"fmt"

	"github.com/czx-lab/czx/agent"
	"github.com/czx-lab/czx/eventbus"
	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/network/jsonx"
	"github.com/czx-lab/czx/network/ws"
)

type (
	Hello struct {
		Name string
	}
	HelloReply struct {
		Msg string
	}
)

func main() {
	gateConf := &agent.GateConf{
		WsServerConf: ws.WsServerConf{
			Addr:            ":8080",
			MaxConn:         100000,
			PendingWriteNum: 1000,
			MaxMsgSize:      4096,
			Timeout:         10,
		},
	}

	processor := jsonx.NewProcessor(network.ProcessorConf{
		LittleEndian: false,
	})

	processor.Register(1, &Hello{})
	processor.Register(2, &HelloReply{})
	processor.RegisterHandler(&HelloReply{}, func(args []any) {
		param := args[1].(*Hello)
		args[0].(network.Agent).WriteWithCode(2, &HelloReply{
			Msg: fmt.Sprintf("Hello %s", param.Name),
		})
	})

	closeFlag := make(chan struct{})
	gateway := agent.NewGate(*gateConf).WithPreConn(func(a network.Agent, phm network.ClientAddrMessage) {
		// Pre-handler message logic here
		fmt.Println("Pre-handler message:", phm)
	}).WithFlag(closeFlag).WithProcessor(processor)

	go func() {
		eventbus.DefaultBus.Subscribe(eventbus.EvtNewAgent, func(message any) {
			fmt.Println(message.(network.Agent).LocalAddr().String())
		})
	}()

	fmt.Printf("Starting websocket server at %s...\n", gateConf.WsServerConf.Addr)
	gateway.Start()
}
