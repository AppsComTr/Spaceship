package main

import (
	"github.com/jinzhu/configor"
	"log"
	"os"
	"os/signal"
	"spaceship/server"
	"syscall"
)

func main()  {

	config := &server.Config{}
	err := configor.Load(config, "config.yml")
	if err != nil {
		log.Panicln("Error while reading configurations from config.yml")
	}
	sessionHolder := server.NewSessionHolder(config)

	_ = server.StartServer(sessionHolder, config)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Startup was completed")

	<-c

}
