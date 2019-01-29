package server

import (
	"context"
	"github.com/mediocregopher/radix/v3"
	"log"
	"time"
)

func Watcher(ctx context.Context, redis *radix.Pool, key string) <- chan int {
	watchChan := make(chan int)

	go func(){
		var result int

		for {
			select {
			case <- ctx.Done():
				close(watchChan)
				log.Println("Watcher closes channel")
				return
			default:
				log.Println("Watcher Default")
				err := redis.Do(radix.Cmd(&result, "SCARD", key))
				if err != nil {
					log.Println("Watcher SCARD err: ", err)
					watchChan <- -1 //means err
					close(watchChan)
					break
				}
				watchChan <- result
				time.Sleep(2 * time.Second)//TODO must be config
			}
		}
	}()

	return watchChan
}