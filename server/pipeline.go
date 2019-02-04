package server

import (
	"github.com/globalsign/mgo"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
	"log"
	"spaceship/socketapi"
)

type Pipeline struct {
	config *Config
	gameHolder *GameHolder
	matchmaker Matchmaker
	sessionHolder *SessionHolder
	jsonProtoMarshler *jsonpb.Marshaler
	jsonProtoUnmarshler *jsonpb.Unmarshaler
	db *mgo.Session
	redis radix.Client
	notification *Notification
}

func NewPipeline(config *Config, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, gameHolder *GameHolder, sessionHolder *SessionHolder, matchmaker Matchmaker, db *mgo.Session, redis radix.Client, notification *Notification) *Pipeline {
	return &Pipeline{
		config: config,
		gameHolder: gameHolder,
		matchmaker: matchmaker,
		sessionHolder: sessionHolder,
		jsonProtoMarshler: jsonProtoMarshler,
		jsonProtoUnmarshler: jsonProtoUnmarshler,
		db: db,
		redis: redis,
		notification: notification,
	}
}

func (p *Pipeline) handleSocketRequests(session Session, envelope *socketapi.Envelope) bool {

	switch envelope.Message.(type) {
	case *socketapi.Envelope_MatchUpdate:

		message := envelope.GetMatchUpdate()

		rs := redsyncradix.New([]radix.Client{p.redis})
		mutex := rs.NewMutex("lock|" + message.GameID)
		if err := mutex.Lock(); err != nil {
			log.Println(err.Error())
		} else {
			defer mutex.Unlock()
		}

		_, err := UpdateGame(p.gameHolder, session, p, message)

		if err != nil {
			log.Println("Error occured while updating game state", err)
			_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
				Code: int32(socketapi.Error_RUNTIME_EXCEPTION),
				Message: "Error occured while updating game state: " + err.Error(),
			}}})
		}

		break
	case *socketapi.Envelope_MatchFind:
		p.matchmakerFind(session, envelope)
	case *socketapi.Envelope_MatchJoin:
		p.matchmakerJoin(session, envelope)
	case *socketapi.Envelope_MatchLeave:
		p.matchmakerLeave(session, envelope)
	default:
		// If we reached this point the envelope was valid but the contents are missing or unknown.
		// Usually caused by a version mismatch, and should cause the session making this pipeline request to close.
		log.Println("Unrecognizable payload received.", envelope)
		_ =session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
			Code:    int32(socketapi.Error_UNRECOGNIZED_PAYLOAD),
			Message: "Unrecognized message.",
		}}})
		return false
	}
	
	return true
	
}