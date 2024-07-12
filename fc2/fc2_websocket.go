package fc2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"nhooyr.io/websocket"
)

var (
	// ErrWebSocketServerDisconnection is returned when the server disconnects.
	ErrWebSocketServerDisconnection = errors.New("server disconnected")
	// ErrWebSocketPaidProgram is returned when the server returns a paid program error.
	ErrWebSocketPaidProgram = errors.New("paid program")
	// ErrWebSocketLoginRequired is returned when the server returns a login required error.
	ErrWebSocketLoginRequired = errors.New("login required")
	// ErrWebSocketMultipleConnection is returned when the server returns a multiple connection error.
	ErrWebSocketMultipleConnection = errors.New("multiple connection error")
	// ErrWebSocketStreamEnded is returned when the server ends the stream.
	ErrWebSocketStreamEnded = errors.New("stream ended")
	// ErrWebSocketEmptyPlaylist is returned when the server does not return a valid playlist.
	ErrWebSocketEmptyPlaylist = errors.New("server did not return a valid playlist")
)

// WebSocket is used to interact with the FC2 WebSocket.
type WebSocket struct {
	*http.Client
	url                 string
	log                 *zerolog.Logger
	healthCheckInterval time.Duration

	msgID    int
	msgMutex sync.Mutex
}

// NewWebSocket creates a new WebSocket.
func NewWebSocket(
	client *http.Client,
	url string,
	healthCheckInterval time.Duration,
) *WebSocket {
	logger := log.With().Str("url", url).Logger()
	w := &WebSocket{
		Client:              client,
		msgID:               1,
		url:                 url,
		log:                 &logger,
		healthCheckInterval: healthCheckInterval,
	}
	return w
}

// Dial connects to the WebSocket server.
func (w *WebSocket) Dial(ctx context.Context) (*websocket.Conn, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "ws.Dial")
	defer span.End()
	// Connect to the websocket server
	conn, _, err := websocket.Dial(ctx, w.url, &websocket.DialOptions{
		HTTPClient: w.Client,
	})
	conn.SetReadLimit(10485760) // 10 MiB
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return conn, nil
}

// GetHLSInformation returns the HLS information.
func (w *WebSocket) GetHLSInformation(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan <-chan *WSResponse,
) (*HLSInformation, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "ws.GetHLSInformation")
	defer span.End()
	msgObj, err := w.sendMessageAndWaitResponse(
		ctx,
		conn,
		"get_hls_information",
		nil,
		msgChan,
		5*time.Second,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var arguments HLSInformation
	if err := json.Unmarshal(msgObj.Arguments, &arguments); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if len(arguments.Playlists) > 0 {
		return &arguments, nil
	}
	return nil, ErrWebSocketEmptyPlaylist
}

// Listen listens for messages from the WebSocket server.
func (w *WebSocket) Listen(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan chan<- *WSResponse,
	commentChan chan<- *Comment,
) error {
	// Start listening for messages from the websocket server
	for {
		msgType, msg, err := conn.Read(ctx)
		if err != nil {
			var closeError websocket.CloseError
			if errors.As(err, &closeError) {
				if closeError.Code == websocket.StatusNormalClosure {
					w.log.Info().Msg("websocket closed cleanly")
					return io.EOF
				}
			}
			return err
		}
		switch msgType {
		case websocket.MessageText:
			w.log.Debug().Str("msg", string(msg)).Msg("ws receive")
			var msgObj WSResponse
			if err := json.Unmarshal(msg, &msgObj); err != nil {
				w.log.Error().Str("msg", string(msg)).Err(err).Msg("failed to decode")
				continue
			}

			switch msgObj.Name {
			case "connect_complete":
				w.log.Info().Msg("ws fully connected")
			case "_response_":
				msgChan <- &msgObj
			case "control_disconnection":
				arguments := &ControlDisconnectionArguments{}
				if err := json.Unmarshal(msgObj.Arguments, arguments); err != nil {
					return err
				}
				switch arguments.Code {
				case 4101:
					return ErrWebSocketPaidProgram
				case 4507:
					return ErrWebSocketLoginRequired
				case 4512:
					return ErrWebSocketMultipleConnection
				default:
					return ErrWebSocketServerDisconnection
				}

			case "publish_stop":
				return ErrWebSocketStreamEnded
			case "comment":
				var arguments CommentArguments
				if err := json.Unmarshal(msgObj.Arguments, &arguments); err != nil {
					return err
				}
				if commentChan != nil {
					comments := arguments.Comments
					for _, comment := range comments {
						commentChan <- &comment
					}
				}
			}

		default:
			w.log.Error().
				Int("type", int(msgType)).
				Str("msg", string(msg)).
				Msg("received unhandled msg type")
		}
	}
}

// HeartbeatLoop sends a heartbeat to keep the ws alive.
//
// The only way to exit the heartbeat loop is to have the WS socket closed or to cancel the context.
func (w *WebSocket) HeartbeatLoop(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan <-chan *WSResponse,
) error {
	queryTicker := time.NewTicker(w.healthCheckInterval)
	defer queryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-queryTicker.C:
			if err := w.heartbeat(ctx, conn, msgChan); err != nil {
				return err
			}
		}
	}
}

// heartbeat message, to be sent every 30 seconds, otherwise the connection will drop
func (w *WebSocket) heartbeat(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan <-chan *WSResponse,
) error {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "ws.heartbeat")
	defer span.End()
	_, err := w.sendMessageAndWaitResponse(ctx, conn, "heartbeat", nil, msgChan, 15*time.Second)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (w *WebSocket) sendMessage(
	ctx context.Context,
	conn *websocket.Conn,
	name string,
	arguments interface{},
	msgID int,
) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Build message
	msgObj := make(map[string]interface{})
	msgObj["name"] = name
	if arguments == nil {
		msgObj["arguments"] = struct{}{}
	} else {
		msgObj["arguments"] = arguments
	}
	w.msgMutex.Lock()
	msgObj["id"] = msgID
	w.msgMutex.Unlock()

	// JSON encode
	msg, err := json.Marshal(msgObj)
	if err != nil {
		return err
	}

	w.log.Debug().Str("msg", string(msg)).Msg("ws send")

	if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
		var closeError websocket.CloseError
		if errors.As(err, &closeError) {
			if closeError.Code == websocket.StatusNormalClosure {
				w.log.Info().Msg("websocket closed cleanly")
				return io.EOF
			}
		}
		return err
	}
	return nil
}

func (w *WebSocket) sendMessageAndWaitResponse(
	ctx context.Context,
	conn *websocket.Conn,
	name string,
	arguments interface{},
	msgChan <-chan *WSResponse,
	timeout time.Duration,
) (*WSResponse, error) {
	defer func() {
		w.msgMutex.Lock()
		w.msgID++
		w.msgMutex.Unlock()
	}()

	// Bump msgID
	w.msgMutex.Lock()
	msgID := w.msgID
	w.msgMutex.Unlock()

	done := make(chan struct{})
	defer func() {
		done <- struct{}{}
		close(done)
	}()
	msgChan = filterMessageByID(done, msgChan, msgID)

	// Send the message
	if err := w.sendMessage(ctx, conn, name, arguments, msgID); err != nil {
		return nil, err
	}

	// Await with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		err := ctx.Err()
		log.Warn().Err(err).Msg("canceled awaiting for response")
		return nil, err
	case msg := <-msgChan:
		return msg, nil
	}
}

func filterMessageByID(
	done <-chan struct{},
	in <-chan *WSResponse,
	expectedID int,
) <-chan *WSResponse {
	out := make(chan *WSResponse, 10)
	go func() {
		defer close(out)
		for {
			select {
			case msg, ok := <-in:
				if !ok {
					return
				}
				if expectedID == msg.ID {
					out <- msg
				}
			case <-done:
				return
			}
		}
	}()
	return out
}
