package server

const (
	GAME_TYPE_PASSIVE_TURN_BASED int = iota
	GAME_TYPE_ACTIVE_TURN_BASED
	GAME_TYPE_REAL_TIME
)

type GameSpecs struct {
	PlayerCount int
	Mode int
}
