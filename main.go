package main

import (
	"log"
	"os"
	"os/signal"
	"spaceship/server"
	"syscall"
)

func main()  {

	_ = server.StartServer()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Startup was completed")

	<-c

}
