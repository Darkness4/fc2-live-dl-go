package fc2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

var (
	ErrWebSocketServerDisconnection = errors.New("server disconnected")
	ErrWebSocketPaidProgram         = errors.New("paid program")
	ErrWebSocketLoginRequired       = errors.New("login required")
	ErrWebSocketMultipleConnection  = errors.New("multiple connection error")
	ErrWebSocketStreamEnded         = errors.New("stream ended")
	ErrWebSocketEmptyPlaylist       = errors.New("server did not return a valid playlist")
)

type WebSocket struct {
	*http.Client
	url                 string
	log                 *zap.Logger
	healthCheckInterval time.Duration

	msgID    int
	msgMutex sync.Mutex
}

func NewWebSocket(
	client *http.Client,
	url string,
	healthCheckInterval time.Duration,
) *WebSocket {
	w := &WebSocket{
		Client:              client,
		msgID:               1,
		url:                 url,
		log:                 logger.I.With(zap.String("url", url)),
		healthCheckInterval: healthCheckInterval,
	}
	return w
}

func (w *WebSocket) Dial(ctx context.Context) (*websocket.Conn, error) {
	// Connect to the websocket server
	conn, _, err := websocket.Dial(ctx, w.url, &websocket.DialOptions{
		HTTPClient: w.Client,
	})
	conn.SetReadLimit(10485760) // 10 MiB
	return conn, err
}

func (w *WebSocket) GetHLSInformation(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan <-chan *WSResponse,
) (*HLSInformation, error) {
	arguments, err := try.DoExponentialBackoffWithResult(
		5,
		2*time.Second,
		2,
		30*time.Second,
		func() (*HLSInformation, error) {
			msgObj, err := w.sendMessageAndWaitResponse(
				ctx,
				conn,
				"get_hls_information",
				nil,
				msgChan,
				5*time.Second,
			)
			if err != nil {
				return nil, err
			}

			var arguments HLSInformation
			if err := json.Unmarshal(msgObj.Arguments, &arguments); err != nil {
				return nil, err
			}
			if len(arguments.Playlists) > 0 {
				return &arguments, nil
			}
			return nil, ErrWebSocketEmptyPlaylist
		},
	)
	if err != nil {
		return nil, err
	}
	return arguments, nil
}

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
					logger.I.Info("websocket closed cleanly")
					return io.EOF
				}
			}
			return err
		}
		switch msgType {
		case websocket.MessageText:
			w.log.Debug("ws receive", zap.String("msg", string(msg)))
			var msgObj WSResponse
			if err := json.Unmarshal(msg, &msgObj); err != nil {
				w.log.Error("failed to decode", zap.Error(err))
			}

			switch msgObj.Name {
			case "connect_complete":
				logger.I.Info("ws fully connected")
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
			w.log.Error(
				"received unhandled msg type",
				zap.Int("type", int(msgType)),
				zap.String("msg", string(msg)),
			)
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
	_, err := w.sendMessageAndWaitResponse(ctx, conn, "heartbeat", nil, msgChan, 15*time.Second)
	return err
}

func (w *WebSocket) sendMessage(
	ctx context.Context,
	conn *websocket.Conn,
	name string,
	arguments interface{},
	msgID int,
) error {
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

	w.log.Debug("ws send", zap.String("msg", string(msg)))

	if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
		var closeError websocket.CloseError
		if errors.As(err, &closeError) {
			if closeError.Code == websocket.StatusNormalClosure {
				logger.I.Info("websocket closed cleanly")
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
		logger.I.Warn("canceled awaiting for response", zap.Error(err))
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
