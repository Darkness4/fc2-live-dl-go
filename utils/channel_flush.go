package utils

import "github.com/rs/zerolog/log"

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
				log.Info().Int("n", count).Msg("flushed messages")
			}
			// No more messages in the channel, so we're done flushing
			return
		}
	}
}
