package server

import (
	"context"
	"github.com/mediocregopher/radix/v3"
	"time"
)

func Watcher(ctx context.Context, redis radix.Client, key string, logger *Logger, config *Config) <- chan int {
	watchChan := make(chan int)

	go func(){
		var result int

		for {
			select {
			case <- ctx.Done():
				close(watchChan)
				logger.Info("Watcher closes channel")
				return
			default:
				err := redis.Do(radix.Cmd(&result, "SCARD", key))
				if err != nil {
					logger.Errorw("Redis error", "command", "SCARD", "key", key, "error", err)
					watchChan <- -1 //means err
					close(watchChan)
					break
				}
				watchChan <- result
				time.Sleep(time.Duration(config.RedisConfig.WatcherIntervalMS) * time.Millisecond)
			}
		}
	}()

	return watchChan
}