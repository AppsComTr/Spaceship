package test

import (
	"bytes"
	"context"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/configor"
	"github.com/mediocregopher/radix/v3"
	"github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"log"
	"net"
	"runtime"
	"spaceship/api"
	"spaceship/apigrpc"
	"spaceship/server"
	"spaceship/socketapi"
	"testing"
)

var (
	jsonpbMarshaler = &jsonpb.Marshaler{
		EnumsAsInts:  true,
		EmitDefaults: false,
		Indent:       "",
		OrigName:     true,
	}
	jsonpbUnmarshaler = &jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}
)

func NewServer(t *testing.T) (*server.Server) {

	config := &server.Config{}
	err := configor.Load(config, "config.yml")
	if err != nil {
		t.Error("Error while reading configurations from config.yml")
	}
	redis := redisConnect(t, config)

	db := server.ConnectDB(config)
	leaderboard := server.NewLeaderboard(db)
	sessionHolder := server.NewSessionHolder(config)
	gameHolder := server.NewGameHolder(redis, jsonpbMarshaler, jsonpbUnmarshaler, leaderboard)
	matchmaker := server.NewLocalMatchMaker(redis, gameHolder, sessionHolder)
	pipeline := server.NewPipeline(config, jsonpbMarshaler, jsonpbUnmarshaler, gameHolder, sessionHolder, matchmaker, db, redis)

	gameHolder.Add(&PTGame{})
	gameHolder.Add(&RTGame{})

	return server.StartServer(sessionHolder, gameHolder, config, jsonpbMarshaler, jsonpbUnmarshaler, pipeline, db, leaderboard)

}

func CreateSession(t *testing.T) (*api.Session) {

	conn, err := grpc.Dial("localhost:7349", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}

	client := apigrpc.NewSpaceShipClient(conn)
	session, err := client.AuthenticateFingerprint(context.Background(), &api.AuthenticateFingerprint{
		Fingerprint: generateUUID(),
	})
	if err != nil {
		t.Fatal(err)
	}

	return session

}

func CreateSessionChan(failChan chan string) (*api.Session) {

	conn, err := grpc.Dial("localhost:7349", grpc.WithInsecure())
	if err != nil {
		failChan <- err.Error()
	}

	client := apigrpc.NewSpaceShipClient(conn)
	session, err := client.AuthenticateFingerprint(context.Background(), &api.AuthenticateFingerprint{
		Fingerprint: generateUUID(),
	})
	if err != nil {
		failChan <- err.Error()
	}

	return session

}

func CreateSocketConn(t *testing.T, token string) (*websocket.Conn, chan []byte) {

	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:7350/ws?token=" + token, nil)
	if err != nil {
		t.Fatal(err)
	}

	onMessageChan := make(chan []byte)

	go func() {
		defer close(onMessageChan)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {

				}else if e, ok := err.(*net.OpError); ok || e.Err.Error() == "use of closed network connection" {

				}else{
					t.Fatal(err)
				}
				//Even if connection was closed or error occured we should break the loop
				break
			}
			onMessageChan <- message
		}
	}()

	return c, onMessageChan

}

func CreateSocketConnChan(failChan chan string, token string) (*websocket.Conn, chan []byte) {

	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:7350/ws?token=" + token, nil)
	if err != nil {
		failChan <- err.Error()
	}

	onMessageChan := make(chan []byte)

	go func() {
		defer close(onMessageChan)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {

				}else if e, ok := err.(*net.OpError); ok || e.Err.Error() == "use of closed network connection" {

				}else{
					failChan <- err.Error()
				}
				//Even if connection was closed or error occured we should break the loop
				break
			}
			onMessageChan <- message
		}
	}()

	return c, onMessageChan

}

func WriteMessage(failChan chan string, client *websocket.Conn, envelope *socketapi.Envelope) {
	var payload []byte
	var err error
	var buf bytes.Buffer

	if err = jsonpbMarshaler.Marshal(&buf, envelope); err == nil {
		payload = buf.Bytes()
	}
	if err != nil {
		failChan <- "Could not marshal envelope " + err.Error()
		runtime.Goexit()
	}

	err = client.WriteMessage(websocket.TextMessage, payload)
	if err != nil {
		failChan <- err.Error()
		runtime.Goexit()
	}
}

func ReadMessage(failChan chan string, onMessageChan chan []byte) (socketapi.Envelope) {
	var payload []byte
	var env socketapi.Envelope

	payload = <- onMessageChan

	if err := jsonpbUnmarshaler.Unmarshal(bytes.NewReader(payload), &env); err != nil {
		log.Println(payload)
		failChan <- err.Error()
		runtime.Goexit()
	}

	return env
}

func redisConnect(t *testing.T, config *server.Config) radix.Client{
	var redisClient radix.Client
	var err error

	if config.RedisConfig.CluesterEnabled {
		redisClient, err = radix.NewCluster([]string{config.RedisConfig.ConnString})
		if err != nil {
			t.Fatal("Redis Connection Failed", err)
		}
	}else{
		redisClient, err = radix.NewPool("tcp", config.RedisConfig.ConnString, 1)
		if err != nil {
			t.Fatal("Redis Connection Failed", err)
		}
	}
	return redisClient
}

func generateUUID() string {
	return uuid.NewV4().String()
}