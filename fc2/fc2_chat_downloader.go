package fc2

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"reflect"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/rs/zerolog/log"
)

// DownloadChat downloads chat messages to a file.
func DownloadChat(
	ctx context.Context,
	commentChan <-chan *api.Comment,
	fName string,
) error {
	log := log.Ctx(ctx)
	file, err := os.Create(fName)
	if err != nil {
		return err
	}

	filteredCommentChannel := removeDuplicatesComment(commentChan)

	// Write to file
	for {
		select {
		case data, ok := <-filteredCommentChannel:
			if !ok {
				log.Error().Msg("writing chat failed, channel was closed")
				return io.EOF
			}
			if data == nil {
				continue
			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				return err
			}
			_, err = file.Write(jsonData)
			if err != nil {
				return err
			}
			_, err = file.Write([]byte("\n"))
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func removeDuplicatesComment(input <-chan *api.Comment) <-chan *api.Comment {
	output := make(chan *api.Comment)
	var last *api.Comment

	go func() {
		defer close(output)
		for new := range input {
			if !reflect.DeepEqual(new, last) {
				output <- new
			}
			last = new
		}
	}()

	return output
}
