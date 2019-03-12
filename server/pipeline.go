package server

import (
	"github.com/globalsign/mgo"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
	"spaceship/model"
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
	logger *Logger
	pubSub *PubSub
}

func NewPipeline(config *Config, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, gameHolder *GameHolder, sessionHolder *SessionHolder, matchmaker Matchmaker, db *mgo.Session, redis radix.Client, notification *Notification, logger *Logger, pubSub *PubSub) *Pipeline {
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
		logger: logger,
		pubSub: pubSub,
	}
}

func (p *Pipeline) handleSocketRequests(session Session, envelope *socketapi.Envelope) bool {

	switch envelope.Message.(type) {
	case *socketapi.Envelope_GameUpdate:

		message := envelope.GetGameUpdate()

		rs := redsyncradix.New([]radix.Client{p.redis})
		mutex := rs.NewMutex("lock|" + message.GameID)
		if err := mutex.Lock(); err != nil {
			p.logger.Errorw("Error while using lock", "key", "lock|" + message.GameID)
		} else {
			defer mutex.Unlock()
		}

		_, err := UpdateGame(p.gameHolder, session, p, message, p.logger, p.matchmaker)

		if err != nil {
			p.logger.Errorw("Error occured while updating game state", "error", err)
			_ = p.pubSub.Send(&model.PubSubMessage{
				UserIDs: []string{session.UserID()},
				Data: &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
					Code: int32(socketapi.Error_RUNTIME_EXCEPTION),
					Message: "Error occured while updating game state: " + err.Error(),
				}}},
			})
			//_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
			//	Code: int32(socketapi.Error_RUNTIME_EXCEPTION),
			//	Message: "Error occured while updating game state: " + err.Error(),
			//}}})
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
		p.logger.Errorw("Unrecognizable payload received.", "envelope", envelope)
		_ = p.pubSub.Send(&model.PubSubMessage{
			UserIDs: []string{session.UserID()},
			Data: &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
				Code:    int32(socketapi.Error_UNRECOGNIZED_PAYLOAD),
				Message: "Unrecognized message.",
			}}},
		})
		//_ =session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
		//	Code:    int32(socketapi.Error_UNRECOGNIZED_PAYLOAD),
		//	Message: "Unrecognized message.",
		//}}})
		return false
	}
	
	return true
	
}