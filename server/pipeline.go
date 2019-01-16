package server

import (
	"github.com/golang/protobuf/jsonpb"
	"log"
	"spaceship/socketapi"
)

type Pipeline struct {
	config *Config
	gameHolder *GameHolder
	jsonProtoMarshler *jsonpb.Marshaler
	jsonProtoUnmarshler *jsonpb.Unmarshaler
}

func NewPipeline(config *Config, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, gameHolder *GameHolder) *Pipeline {
	return &Pipeline{
		config: config,
		gameHolder: gameHolder,
		jsonProtoMarshler: jsonProtoMarshler,
		jsonProtoUnmarshler: jsonProtoUnmarshler,
	}
}

func (p *Pipeline) handleSocketRequests(session Session, envelope *socketapi.Envelope) bool {

	switch envelope.Message.(type) {
	case *socketapi.Envelope_MatchStart:

		log.Println("Match start message was retrieved id : " + session.ID().String())
		message := envelope.GetMatchStart()

		game := p.gameHolder.Get(message.GameName)

		if game == nil {
			session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
				Code:    int32(socketapi.Error_UNRECOGNIZED_PAYLOAD),
				Message: "Unrecognized message for given match start request.",
			}}})
		}else{

			game.Start(session)

		}
		break
	default:
		// If we reached this point the envelope was valid but the contents are missing or unknown.
		// Usually caused by a version mismatch, and should cause the session making this pipeline request to close.
		log.Println("Unrecognizable payload received.", envelope)
		session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
			Code:    int32(socketapi.Error_UNRECOGNIZED_PAYLOAD),
			Message: "Unrecognized message.",
		}}})
		return false
	}
	
	return true
	
}