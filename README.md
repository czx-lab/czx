## Czx 游戏服务器框架简介
---------------
该框架的network、timer、modeule部分参照Leaf框架修改而来。
框架还包含player、room、eventbus、以及帧同步等组件，让服务端尽可能的提升开发的效率。

### 模块机制
---------------
每个模块都运行在一个单独的 goroutine 中，框架模块之间的通信目前未设计，可通过`eventbus`来进行异步的通信。
模块用法如下：
- 注册
```go
// 注册一个游戏模块
type Game struct {}

// 实现czx.Module接口
var _ czx.Module = (*Game)(nil)

// 注册模块
czx.Register(&Game{})
```

- 初始化并运行
```go
czx.Init()
```

- 清理所有已注册的模块，并等待它们完成
```go
czx.Destroy()
```

每个模块都需要实现 czx.Module 接口：
```go
type Module interface {
	Init()
	Destroy()
	Run(done chan struct{})
}
```

也可直接使用 `czx.Run`完成上面三个流程：
```go
type Game struct {}
var _ czx.Module = (*Game)(nil)

mods := []czx.Module{
    &Game{},
}
czx.Run(mods...)
```

Czx 首先会在同一个 goroutine 中按模块注册顺序执行模块的 Init 方法，等到所有模块 Init 方法执行完成后则为每一个模块启动一个 goroutine 并执行模块的 Run 方法。最后，游戏服务器关闭时（Ctrl + C 关闭游戏服务器）将按模块注册相反顺序在同一个 goroutine 中执行模块的 Destroy 方法。

### Czx 源码概览
---------------
- czx/eventbus 提供了事件驱动的方式来解耦业务逻辑
- czx/frame 消息 loop 同步相关, 负责循环推送消息与接收消息，帧同步与普通同步
- czx/gnetx 基于 [gnet](https://github.com/panjf2000/gnet) 的 network 的扩展
- czx/network 网络相关，使用 TCP 和 WebSocket 协议，可自定义消息格式，默认 Czx 提供了基于 protobuf 和 JSON 的消息格式
- czx/player 玩家相关，提供了玩家管理、玩家心跳等
- czx/room 提供房间的创建、管理，以及房间玩家匹配机制
- czx/timer 定时器相关
- czx/xlog 日志相关