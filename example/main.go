package main

import (
	"fmt"

	"github.com/czx-lab/czx/agent"
	"github.com/czx-lab/czx/eventbus"
	"github.com/czx-lab/czx/example/pb"
	"github.com/czx-lab/czx/network"
	"github.com/czx-lab/czx/network/protobuf"
	"github.com/czx-lab/czx/network/ws"
)

type (
	Hello struct {
		Name string
	}
	HelloReply struct {
		Msg string
	}

	HelloCopy struct {
		Name string
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

	// processor := jsonx.NewProcessor(network.ProcessorConf{
	// 	LittleEndian: false,
	// })
	// processor.Register(network.Message{
	// 	Data: &Hello{},
	// })
	// processor.Register(network.Message{
	// 	Data: &HelloCopy{},
	// })
	// processor.Register(network.Message{
	// 	Data: &HelloReply{},
	// })
	// processor.RegisterHandler(&HelloCopy{}, func(args []any) {
	// 	param := args[0].(*HelloCopy)
	// 	args[1].(network.Agent).WriteWithCode(201, &HelloReply{
	// 		Msg: fmt.Sprintf("Hello copy %s", param.Name),
	// 	})
	// })

	// processor.RegisterHandler(&Hello{}, func(args []any) {
	// 	param := args[0].(*Hello)
	// 	args[1].(network.Agent).WriteWithCode(201, &HelloReply{
	// 		Msg: fmt.Sprintf("Hello %s", param.Name),
	// 	})
	// })

	processor := protobuf.NewProcessor(network.ProcessorConf{
		LittleEndian: false,
	})
	processor.Register(network.Message{
		ID:   1000,
		Data: &pb.HelloReq{},
	})
	processor.Register(network.Message{
		ID:   1001,
		Data: &pb.HelloResp{},
	})
	processor.RegisterHandler(&pb.HelloReq{}, func(args []any) {
		param := args[0].(*pb.HelloReq)
		args[1].(network.Agent).WriteWithCode(200, &pb.HelloResp{
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
			fmt.Println("LocalAddr = ", message.(network.Agent).LocalAddr().String())
		})
	}()

	fmt.Printf("Starting websocket server at %s...\n", gateConf.WsServerConf.Addr)
	gateway.Start()
}
