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
		EmitDefaults: true,
		Indent: "",
		OrigName: true,
	}
	jsonProtoUnmarshler = &jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}
)

func main() {

	config := &server.Config{}
	err := configor.Load(config, "config.yml")
	if err != nil {
		log.Panicln("Error while reading configurations from config.yml")
	}

	logger := server.NewLogger(config)
	defer logger.Sync()

	redis := redisConnect(config, logger)

	db := server.ConnectDB(config, logger)
	notification := server.NewNotificationService(db, config, logger)
	leaderboard := server.NewLeaderboard(db, logger)
	stats := server.NewStatsHolder(logger)
	sessionHolder := server.NewSessionHolder(config)
	gameHolder := server.NewGameHolder(redis, jsonProtoMarshaler, jsonProtoUnmarshler, leaderboard, notification)
	leaderboard.SetGameHolder(gameHolder)
	matchmaker := server.NewLocalMatchMaker(redis, gameHolder, sessionHolder, notification, logger)
	pipeline := server.NewPipeline(config, jsonProtoMarshaler, jsonProtoUnmarshler, gameHolder, sessionHolder, matchmaker, db, redis, notification, logger)

	sessionHolder.SetLeaveListener(matchmaker.LeaveActiveGames)

	initGames(gameHolder)

	server := server.StartServer(sessionHolder, gameHolder, config, jsonProtoMarshaler, jsonProtoUnmarshler, pipeline, db, leaderboard, stats, logger)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Startup was completed")

	<-c

	logger.Info("Shutdown was started")
	server.Stop()
	logger.Info("Shutdown was completed")

	os.Exit(1)

}

func initGames(holder *server.GameHolder) {
	holder.Add(&game.ExampleGame{})
	holder.Add(&game.ExampleATGame{})
	holder.Add(&game.RTGame{})
}

func redisConnect(config *server.Config, logger *server.Logger) radix.Client{

	var redisClient radix.Client
	var err error

	if config.RedisConfig.CluesterEnabled {
		redisClient, err = radix.NewCluster([]string{config.RedisConfig.ConnString})
		if err != nil {
			logger.Fatalw("Redis Connection Failed", "error", err)
		}
	}else{
		redisClient, err = radix.NewPool("tcp", config.RedisConfig.ConnString, 1)
		if err != nil {
			logger.Fatalw("Redis Connection Failed", "error", err)
		}
	}
	return redisClient
}