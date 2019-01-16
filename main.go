package main

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/jinzhu/configor"
	"log"
	"os"
	"os/signal"
	"spaceship/game"
	"spaceship/server"
	"syscall"
)

var (
	jsonProtoMarshaler = &jsonpb.Marshaler{
		EnumsAsInts: true,
		EmitDefaults: false,
		Indent: "",
		OrigName: true,
	}
	jsonProtoUnmarshler = &jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}
)

func main()  {

	config := &server.Config{}
	err := configor.Load(config, "config.yml")
	if err != nil {
		log.Panicln("Error while reading configurations from config.yml")
	}
	sessionHolder := server.NewSessionHolder(config)
	gameHolder := server.NewGameHolder()
	pipeline := server.NewPipeline(config, jsonProtoMarshaler, jsonProtoUnmarshler, gameHolder)

	initGames(gameHolder)

	server := server.StartServer(sessionHolder, gameHolder, config, jsonProtoMarshaler, jsonProtoUnmarshler, pipeline)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Startup was completed")

	<-c

	log.Println("Shutdown was started")
	server.Stop()
	log.Println("Shutdown was completed")

	os.Exit(1)

}

func initGames(holder *server.GameHolder) {
	holder.Add(&game.TestGame{})
	holder.Add(&game.TestSecondGame{})
}
