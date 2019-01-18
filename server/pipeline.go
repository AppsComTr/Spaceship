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
			_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
				Code:    int32(socketapi.Error_UNRECOGNIZED_PAYLOAD),
				Message: "Unrecognized message for given match start request.",
			}}})
		}else{

			//Matchmaker will process this side
			//game.Create()

		}
		break
	case *socketapi.Envelope_MatchUpdate:

		message := envelope.GetMatchUpdate()
		gameController := p.gameHolder.Get(message.GameName)

		if gameController == nil {
			log.Println("Game controller couldn't find with given game name", envelope)
			_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
				Code:    int32(socketapi.Error_BAD_INPUT),
				Message: "Game controller couldn't find with given game name",
			}}})
		}

		_, err := UpdateGame(session, message, gameController.Update)

		if err != nil {
			log.Println("Error occured while updating game state", err)
			_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
				Code: int32(socketapi.Error_RUNTIME_EXCEPTION),
				Message: "Error occured while updating game state",
			}}})
		}

		break
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