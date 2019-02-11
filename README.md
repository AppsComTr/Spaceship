#Spaceship

> Multiplayer game backend framework

#Features

- Authentication support with fingerprint and Facebook account
- Built-in friendship mechanism
- Passive turn-based, active turn-based and real time game support
- Leaderboard with built-in groups like (daily, weekly etc. and for spececific game or overall the games)
- Matchmaker with customizable parameters for all game types
- Simple metrics which is collected with OpenCensus

You can build your own game server with just working on logic of the game. We don't support vertical scaling for now.

##Getting started

These instructions will get you a copy of the project up and running on your local machine for development and testing.

### Prerequisites

Spaceship requires Redis and MongoDB. 
Redis is generally used by matchmaker module and storing game datas which is not finished yet to speed up processes.
MongoDB is used for persistancy of user datas, scores and finished game datas etc...

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

We've already prepared example games of all game types and test codes. These files can be found under `test` package.
These tests simulates clients for designed games. If you start up Spaceship correctly, tests should be successful.

## Development with Spaceship

As can be seen in the feature screen, developers should only focus on developing their own game logic. 
Spaceship handles all other things for them. Spaceship allows defining multiple games.
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
	
- `GetGameSpecs()` method:

	This method is used by Spaceship to define this game's specifications. This method should return a `GameSpecs` struct. This struct is consist of 3 field for now:
	
	```go
	type GameSpecs struct {
    	PlayerCount int
    	Mode int
    	TickInterval int
    }
	```
	
	`PlayerCount` is used by match maker module. This field should contains the number of maximum player count for this game.
	
	`Mode` defines the type of this game. We support 3 game types. These are real time, active turn based and passive turn based. This fields value should be one of the predefined constants in Spaceship.
	```go
	const (
    	GAME_TYPE_PASSIVE_TURN_BASED int = iota
    	GAME_TYPE_ACTIVE_TURN_BASED
    	GAME_TYPE_REAL_TIME
    )
	```
	
	`TickInterval` is optional. If designed game is a real time game, should contains a valid value in ms format. This is used to define interval between running of game loops. 