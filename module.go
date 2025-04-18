package czx

import (
	"czx/xlog"
	"runtime"
	"sync"

	"go.uber.org/zap"
)

var (
	defaultIsStackBuf  = false   // Whether to print stack information when an error occurs
	defaultStackBufLen = 4096    // Length of the stack buffer
	mods               []*module // List of registered modules
)

type (
	ModuleConf struct {
		IsStackBuf  bool // Whether to print stack information when an error occurs
		StackBufLen int  // Length of the stack buffer
	}
	// Module interface defines the structure for a module in the system.
	// It includes methods for initialization, destruction, and running the module.
	Module interface {
		// Init initializes the module and prepares it for use.
		// This method should be called before using any other methods of the module.
		Init()
		// Destroy cleans up resources used by the module.
		Destroy()
		// Run starts the module's main loop or process.
		Run(done chan struct{})
	}

	module struct {
		mi  Module
		wg  sync.WaitGroup
		sig chan struct{}
	}
)

// MustConf sets the default configuration for modules.
// It should be called before initializing any modules.
func MustConf(conf ModuleConf) {
	defaultIsStackBuf = conf.IsStackBuf
	defaultStackBufLen = conf.StackBufLen
}

// Register registers a module to be managed by the system.
// It should be called before initializing any modules.
func Register(mi Module) {
	m := new(module)
	m.mi = mi
	m.sig = make(chan struct{}, 1)

	mods = append(mods, m)
}

// Init initializes all registered modules and starts them in separate goroutines.
// It should be called before starting any modules.
func Init() {
	// Initialize all registered modules.
	// This method should be called before starting any modules.
	for i := range mods {
		mods[i].mi.Init()
	}

	// Start all registered modules in separate goroutines.
	// This method should be called to start the modules' main loops.
	for i := range mods {
		m := mods[i]
		m.wg.Add(1)
		go run(m)
	}
}

// Destroy cleans up all registered modules and waits for their completion.
// It should be called when the application is shutting down.
func Destroy() {
	for i := len(mods) - 1; i >= 0; i-- {
		m := mods[i]
		m.sig <- struct{}{}
		m.wg.Wait()
		destroy(m)
	}
}

// run starts the module's main loop and waits for its completion.
// It should be called in a separate goroutine for each module.
func run(m *module) {
	m.mi.Run(m.sig)
	m.wg.Done()
}

// destroy cleans up the resources used by the module and handles any errors that may occur during the cleanup process.
// It should be called when the module is no longer needed.
func destroy(m *module) {
	defer func() {
		if r := recover(); r != nil {
			if defaultIsStackBuf {
				buf := make([]byte, defaultStackBufLen)
				l := runtime.Stack(buf, false)
				xlog.Write().Sugar().Errorf("%v: %s", r, buf[:l])
			} else {
				xlog.Write().Error("module destroy panic: ", zap.Any("module panic", r))
			}
		}
	}()

	m.mi.Destroy()
}
