package server

import (
	"spaceship/socketapi"
)

func (p *Pipeline) matchmakerFind(session Session, envelope *socketapi.Envelope){
	incomingData := envelope.GetMatchFind()
	matchEntry, err := p.matchmaker.Find(session, incomingData.GameName, incomingData.QueueProperties)
	if err != nil {
		_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
			Code:    int32(socketapi.Error_MATCH_JOIN_REJECTED),
			Message: "Could not find match.",
		}}})
	}

	_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_MatchEntry{MatchEntry:matchEntry}})
}

func (p *Pipeline) matchmakerJoin(session Session, envelope *socketapi.Envelope){
	incomingData := envelope.GetMatchJoin()
	game, err := p.matchmaker.Join(p, session, incomingData.MatchId)
	if err != nil {
		_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_Error{Error: &socketapi.Error{
			Code:    int32(socketapi.Error_MATCH_JOIN_REJECTED),
			Message: "Could not join match.",
		}}})
	}

	ms := socketapi.MatchJoinResp{GameData: game}
	_ = session.Send(false, 0, &socketapi.Envelope{Cid: envelope.Cid, Message: &socketapi.Envelope_MatchStart{MatchStart:&ms}})
}

func (p *Pipeline) matchmakerLeave(session Session, envelope *socketapi.Envelope){
	incomingData := envelope.GetMatchLeave()
	err := p.matchmaker.Leave(session, incomingData.MatchId)
	if err != nil {
		p.logger.Errorw("Error in matchmakers leave method", "error", err)
	}
}