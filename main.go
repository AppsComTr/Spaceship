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
	"github.com/mediocregopher/radix/v3"
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
	redis := redisConnect(config)

	sessionHolder := server.NewSessionHolder(config)
	gameHolder := server.NewGameHolder(redis, jsonProtoMarshaler, jsonProtoUnmarshler)
	matchmaker := server.NewLocalMatchMaker(redis, gameHolder)
	pipeline := server.NewPipeline(config, jsonProtoMarshaler, jsonProtoUnmarshler, gameHolder, matchmaker)

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
}


func redisConnect(config *server.Config) *radix.Pool{
	redisPool, err := radix.NewPool("tcp", config.RedisConfig.ConnString, 1)
	if err != nil {
		log.Fatalln("Redis Connection Failed", err)
	}
	return redisPool
}