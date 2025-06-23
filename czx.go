package czx

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

var version = "1.6.1"

// Module represents a module in the czx framework.
func Run(mods ...Module) {
	log.Printf("Czx %v starting up \n", version)

	for i := range mods {
		Register(mods[i])
	}

	Init()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	log.Printf("Czx shutting down (signal: %v)", sig)

	Destroy()
}
