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
	RedisConfig struct{
		ConnString string `default:"redis"`
		CluesterEnabled bool `default:"false"`
	}
	AuthConfig struct{
		JWTSecret string `default:"asdasdqweqasdqwwe"`
		TokenExpireTime int `default:"86400"`
	}
	Port int `default:"7350"`
	ApiURL string `default:"http://localhost"`
	MaxRequestBodySize int64 `default:"4096"`
	DevelopmentEnabled bool `default:"false"`
	NotificationConfig struct{
		AppKey string
		AppID string
	}
}
