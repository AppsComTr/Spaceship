package server

type Config struct {
	SocketConfig struct{
		PingPeriodTime int `default:"8000"`
		PongWaitTime int `default:"10000"`
		WriteWaitTime int `default:"5000"`
		ReceivedMessageDecrementCount int `default:"20"`
		OutgoingQueueSize int `default:"64"`
	}
	DBConfig struct{
		ConnString string `default:"mongo"`
	}
	AuthConfig struct{
		JWTSecret string `default:"asdasdqweqasdqwwe"`
		TokenExpireTime int `default:"86400"`
	}
	Port int `default:"7350"`
}
