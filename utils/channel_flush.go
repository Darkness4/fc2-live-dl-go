package utils

import (
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"go.uber.org/zap"
)

func Flush[T any](msgChan chan T) {
	count := 0
	for {
		select {
		case _, ok := <-msgChan:
			if !ok {
				return
			}
			count++
		default:
			if count != 0 {
				logger.I.Info("flushed messages", zap.Int("n", count))
			}
			// No more messages in the channel, so we're done flushing
			return
		}
	}
}
