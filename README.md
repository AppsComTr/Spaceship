## Spaceship

> Multiplayer game backend framework

## Features

- Authentication support with fingerprint and Facebook account
- Built-in friendship mechanism
- Passive turn-based, active turn-based and real time game support
- Leaderboard with built-in groups like (daily, weekly etc. and for spececific game or overall the games)
- Matchmaker with customizable parameters for all game types
- Simple metrics which is collected with OpenCensus
- Distributed system support with message broker ([RabbitMQ](https://www.rabbitmq.com))

Spaceship is an online game backend framework designed to allow game creators to build their own game's server side without the hassle of the common parts of every multiplayer games.
You can build your own game server with just working on logic of the game. We have support for distributed systems now.

Spaceship has no client-side library for now. But currently we are working on **Unity** client.

## Getting started

These instructions will help you to get a copy of the project and run on your local machine for development and testing.

### Prerequisites

Spaceship requires **Redis**, **MongoDB** and **RabbitMQ**_(optional)_.

- Redis is generally used by matchmaker module and storing game datas which is not finished yet to speed up processes.
- MongoDB is used for persistancy of user datas, scores and finished game datas etc...
- If you are planning to use Spaceship on distributed system you also need a RabbitMQ server. RabbitMQ is used to publish messages between system nodes and subscribe them on the nodes. But this is optional as stated above. If you don't want to use this feature, just delete connection string for **RabbitMQ** or leave it empty in configuration file. You can follow the instructions from [here](https://www.rabbitmq.com/download.html) to install your own **RabbitMQ** server.

### Installing

You can easily run Spaceship on your local machine with [Docker Compose](https://docs.docker.com/compose/).
Before create and start containers, you just need to build the images first.

```shell
docker-compose build
docker-compose up
```

Or, you can start up Redis and MongoDB server manually, and start server with `go run *.go` command after updating the `config.yml` file.

If you want to modify the Proto files and regenerate auto-genereated files you need to run these commands under the root directory of this project.

```shell
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --go_out=plugins=grpc:. ./api/api.proto
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --grpc-gateway_out=logtostderr=true:. ./api/api.proto
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --go_out=plugins=grpc:. ./apigrpc/apigrpc.proto
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --grpc-gateway_out=logtostderr=true:. ./apigrpc/apigrpc.proto
protoc -I/usr/local/include -I. -I$GOPATH/src -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway --go_out=plugins=grpc:. ./socketapi/socketapi.proto
```

These commands will generate all necessary files for `api`, `apigrpc` and `socketapi` packages.
Or you can just run `make protogen` command.

## Running the tests

We've already prepared simple example games for all game types and test codes. These files can be found under `test` package.
These tests simulates clients for designed games. If you start up Spaceship correctly, tests should be successful.

## Development with Spaceship

As it can be seen in the features section, developers should only focus on developing their own game logic. 
Spaceship handles all other things for them. Spaceship allows defining multiple games on a single server. 

First of all, Spaceship supports 3 types of games. These are; real time, active turn-based and passive turn-based. 
Real time games works with looper mechanism on server side. Data is not directly processed, they are queued to be handled in loop method with given interval which can be defined in config.
For the other types, datas are not queued and processed in update method when they are arrived to server.
Matchmaker works different for passive turn-based games according to others. Because others are considered as active games and they should be started after complete the user count for given game.
Because of these, `Mode` field in game specs should be defined carefully according to designed game.

To attach a game to Spaceship, developers should implement the `GameController` methods. These are:

```go
type GameController interface {
	GetName() string
	Init(gameData *socketapi.GameData, logger *Logger) error
	Join(gameData *socketapi.GameData, session Session, notification *Notification, logger *Logger) error
	Leave(gameData *socketapi.GameData, session Session, logger *Logger) error
	Update(gameData *socketapi.GameData, session Session, metadata string, leaderboard *Leaderboard, notification *Notification, logger *Logger) (bool, error)
	Loop(gameData *socketapi.GameData, queuedDatas []socketapi.GameUpdateQueue, leaderboard *Leaderboard, notification *Notification, logger *Logger) bool
	GetGameSpecs() GameSpecs
}
```

- `GameName()` method:
	
	Every game should have a unique name. When using client, this names will be used to differenciate the games. So, this method should return static string for this game.
	
- `Init(gameData *socketapi.GameData, logger *Logger) error` method:

	This method will be called by Spaceship when a new game is created. Game specific datas can be set in this method. For example, you can set default state of board if you are developing a puzzle game.
	
	Every game in Spaceship have common struct which is `GameData`. Game data is passed to this method with `gameData` parameter. This struct is generally managed by Spaceship. Developers should store or update game specific datas in `metadata` field of game data.
	
	In Spaceship, we developed logger module to log events or errors. This module is also passed to this method. So, it can be used if it is required.
	
- `Join(gameData *socketapi.GameData, session Session, notification *Notification, logger *Logger) error` method:

	This method will be triggered by Spaceship when a user joins to this game. Matchmaker module decides the users which will join to this game and pass them to this method.
	
	In addition to the previous method, session and notification parameters are also passed to this method. A `Session` stores everything about the connected client. Developers can access user id or session id with this and `Notification` module can be used to send notification other users to inform them about the opponent.
	
- `Leave(gameData *socketapi.GameData, session Session, logger *Logger) error` method:

	When users disconnect from the server or leave the game willingly, this method is called by Spaceship. According to game logic, developers can perform the necessary operations.

- `Update(gameData *socketapi.GameData, session Session, metadata string, leaderboard *Leaderboard, notification *Notification, logger *Logger) (bool, error)` method:

	This method is triggered by Spaceship when clients send data about the game if game mode is turn-based. This can be leaved empty for real time games. For example; if you design turn-based puzzle game, users data should be sent to the server by the client. When Spaceship receive this data, triggers relevant game controllers update method. In this way, any logic for designed game can be executed.
	
	`metadata` contains data which is sent by the client. Developers should build their own game data structs and use any serialization methods that they want. Spaceship only accepts strings for metadata. For example; client send data after serializing game data to json string and can unmarshal it in update method.
	
	Leaderboard module is also passed to this method. So, developers can update user scores according to their game play datas.
	
	Also developers should decide if the game is finished or not in this method and if the game is completed, should return true. So Spaceship can understand that this game is completed and write this game's data to db to make it persistent.
	
- `Loop(gameData *socketapi.GameData, queuedDatas []socketapi.GameUpdateQueue, leaderboard *Leaderboard, notification *Notification, logger *Logger) bool` method:

	As stated above, this method is repeatedly triggered by Spaceship only when the game mode is real time with given interval. Metadatas that comes from clients are passed to this method as array by keeping the arriving order.
	
- `GetGameSpecs()` method:

	This method is used by Spaceship to define this game's specifications. This method should return a `GameSpecs` struct. This struct is consist of 3 field for now:
	
	```go
	type GameSpecs struct {
    	PlayerCount int
    	Mode int
    	TickInterval int
    }
	```
	
	`PlayerCount` is used by match maker module. This field should contain the number of maximum player count for this game.
	
	`Mode` defines the type of this game. As stated above, Spaceship supports 3 game types. These are real time, active turn based and passive turn based. These fields value should be one of the predefined constants in Spaceship. This should be selected carefully according to designed game.
	```go
	const (
    	GAME_TYPE_PASSIVE_TURN_BASED int = iota
    	GAME_TYPE_ACTIVE_TURN_BASED
    	GAME_TYPE_REAL_TIME
    )
	```
	
	`TickInterval` is optional. If designed game is a real time game, should contains a valid value in ms format. This is used to define interval between running of game loops. 
	
Here you can see an example very basic real time game. This game is designed for two players. Players can attack the boss concurrently. When the boss monster is killed, the user hit last wins and the game is finished.

```go
type RTGame struct {}

var rtGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_REAL_TIME,
	TickInterval: 1000,
}

const (
	RT_GAME_STATE_CONTINUE = iota
	RT_GAME_STATE_FINISHED
)

type RTGameUpdateData struct {
	Damage int
}

//Dummy struct for this example game
type RTGameData struct {
	GameState int
	BossHealth int
	WinnerUserID *string
}

func (tg *RTGame) GetName() string {
	//These value should be unique for each games
	return "realtimeTestGame"
}

func (tg *RTGame) Init(gameData *socketapi.GameData, logger *server.Logger) error {

	rtGameData := RTGameData{
		GameState: RT_GAME_STATE_CONTINUE,
		BossHealth: 300,
	}

	data, err := json.Marshal(rtGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(data)

	return nil
}

func (tg *RTGame) Join(gameData *socketapi.GameData, session server.Session, notification *server.Notification, logger *server.Logger) error {

	return nil

}

func (tg *RTGame) Leave(gameData *socketapi.GameData, session server.Session, logger *server.Logger) error {

	return nil
}

func (tg *RTGame) Update(gameData *socketapi.GameData, session server.Session, metadata string, leaderboard *server.Leaderboard, notification *server.Notification, logger *server.Logger) (bool, error) {
	return false, nil
}

func (tg *RTGame) Loop(gameData *socketapi.GameData, queuedDatas []socketapi.GameUpdateQueue, leaderboard *server.Leaderboard, notification *server.Notification, logger *server.Logger) bool {

	var rtGameData RTGameData
	err := json.Unmarshal([]byte(gameData.Metadata), &rtGameData)
	if err != nil {
		logger.Error(err)
		return true
	}

	isFinished := false
	for _, queueItem := range queuedDatas {

		var updateData RTGameUpdateData
		err = json.Unmarshal([]byte(queueItem.Metadata), &updateData)
		if err != nil {
			logger.Error(err)
			return true
		}

		rtGameData.BossHealth -= updateData.Damage

		if rtGameData.BossHealth <= 0 {
			rtGameData.BossHealth = 0
			rtGameData.GameState = RT_GAME_STATE_FINISHED
			rtGameData.WinnerUserID = &queueItem.UserID
			isFinished = true
		}

	}

	rtGameDataS, err := json.Marshal(rtGameData)
	if err != nil {
		return true
	}
	gameData.Metadata = string(rtGameDataS)

	return isFinished

}

func (tg RTGame) GetGameSpecs() server.GameSpecs {
	return rtGameSpecs
}
```
	
## Technical documentation

This simple documentation part is prepared for developers who may want to contribute Spaceship.

While developing Spaceship, we were inspired from [Nakama](https://github.com/heroiclabs/nakama) and [Open Match](https://github.com/GoogleCloudPlatform/open-match). You can also check these projects too. If you want to learn more detailed informations about the project you can [contact us](mailto:mehmet@apps.com.tr).

Spaceship is using rpc technology to serve their endpoints by using [gRPC](https://grpc.ip) framework. It also supports http requests with using [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).
Spaceship allows clients to make request with gzip compressed body over the http.
Also, it supports both json and proto messages over the socket connection. 
As spaceship uses gRPC, [Protocol Buffers](https://github.com/protocolbuffers/protobuf) is used to model messages and services. If you want to add a new service or modify existing one, you should regenerate codes with described in [Installing](#installing) section.

Every single logic part is designed as a module in Spaceship. There are 9 main module in Spaceship. These are:

- Notification
- Leaderboard
- Stats
- Session holder
- Game holder 
- Matchmaker
- Pipeline
- PubSub
- Server

As you can understand from their names, they are only responsible for their own jobs. Modules are initialized with other modules if necessary. 

To start up server, these all modules should be created and after that, server's `StartServer` method should be called with passing all of these modules.
In this method, gRPC and grpc-gateway servers are configured. Additional and necessary endpoints are defined on a router to accept web socket connections and serving metrics for [Prometheus](https://prometheus.io) over an endpoint.
Also one more endpoint is defined on this router to serve static files. For now Spaceship does not support cloud storage.

All other services are defined on the gRPC server instance. They could be found in `api_account.go` and `api_leaderboard.go` files under the server package. 

To accept socket connections, `NewSocketAcceptor` method is used on the router. When a client wants to open a new socket connection, this method is called and it returns an http handler.
This handler first checks if given token is valid, if it is valid, tries to upgrade the http connection to web socket connection and creates `Session`.
Sessions are unique for every user. It holds users active connection, informations, connection states etc... It handles incoming and outgoing messages over the socket connection first with `Consume` and `processOutgoing` methods.

Incoming messages over the socket connection are passed to **pipeline** module. This module is responsible for redirecting message to correct place and returning their responses - if exists - to clients. Game data is broadcast over this module.
For example, when a client sends a match find message, it redirects this message to **matchmaker** module.

**PubSub** module is used to send messages to socket clients. To support horizontal scaling for the projects which use socket connections, a message brokers should be used. Because, when a client triggers something on any node of server that broadcasts messages to other users, other nodes also should be notified. Some of the users may connected to different nodes with redirecting of load balancers.
This module basically subscribe to a queue on the message broker. When a message comes from that queue it checks the user ids which are in the message from session holder to detect sessions that connected to current node. If session exist on this node, data of the message is sent to this client over web socket. Also, this module has a method named `Send`. This method publishes message to all queues. So, subscribers can be notified about the messages. If a message is needed to be sent to the clients, this module's `Send` method should be used.
If connection string was defined for **Rabbit MQ** server, distributed system is supported. Otherwise, it just redirects messages to sessions directly. 
 
 
 ## License
 
 MIT License

Copyright (c) 2019 Apps

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
