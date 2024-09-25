package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

	// ErrQualityNotAvailable is returned when the quality is not available.
	ErrQualityNotAvailable = errors.New("requested quality is not available")
)

var tracerName = "fc2/api"

// WebSocket is used to interact with the FC2 WebSocket.
type WebSocket struct {
	*http.Client
	url                 string
	log                 *zerolog.Logger
	healthCheckInterval time.Duration

	msgID atomic.Int64
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
		url:                 url,
		log:                 &logger,
		healthCheckInterval: healthCheckInterval,
	}
	w.msgID.Add(1)
	return w
}

// Dial connects to the WebSocket server.
func (w *WebSocket) Dial(ctx context.Context) (*websocket.Conn, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "ws.Dial", trace.WithAttributes(
		attribute.String("url", w.url),
	))
	defer span.End()
	// Connect to the websocket server
	conn, _, err := websocket.Dial(ctx, w.url, &websocket.DialOptions{
		HTTPClient: w.Client,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	conn.SetReadLimit(10485760) // 10 MiB
	return conn, nil
}

// GetHLSInformation returns the HLS information.
func (w *WebSocket) GetHLSInformation(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan <-chan *WSResponse,
) (HLSInformation, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "ws.GetHLSInformation", trace.WithAttributes(
		attribute.String("url", w.url),
	))
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
		return HLSInformation{}, err
	}
	if msgObj == nil {
		err := errors.New("no message received")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return HLSInformation{}, err
	}

	var arguments HLSInformation
	if err := json.Unmarshal(msgObj.Arguments, &arguments); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return HLSInformation{}, err
	}
	if len(arguments.Playlists) > 0 {
		return arguments, nil
	}
	return HLSInformation{}, ErrWebSocketEmptyPlaylist
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
		var msgObj WSResponse
		err := wsjson.Read(ctx, conn, &msgObj)
		if err != nil {
			var closeError websocket.CloseError
			if errors.As(err, &closeError) {
				if closeError.Code == websocket.StatusNormalClosure {
					w.log.Info().Msg("websocket closed cleanly")
					return io.EOF
				}
			} else if errors.Is(err, net.ErrClosed) {
				w.log.Info().Msg("websocket closed")
				return io.EOF
			}
			return err
		}
		w.log.Trace().Stringer("msg", msgObj).Msg("ws receive")

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
				w.log.Info().Msg("ws paid program")
				return ErrWebSocketPaidProgram
			case 4507:
				w.log.Info().Msg("ws login required")
				return ErrWebSocketLoginRequired
			case 4512:
				w.log.Info().Msg("ws multiple connection")
				return ErrWebSocketMultipleConnection
			default:
				w.log.Error().Msg("ws server disconnection")
				return ErrWebSocketServerDisconnection
			}

		case "publish_stop":
			w.log.Info().Msg("ws stream ended")
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
	// We ignore the context.DealineExceeded error as failing to receive a heartbeat response is not fatal. (Sending is more important to keep the connection alive.)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

func (w *WebSocket) sendMessage(
	ctx context.Context,
	conn *websocket.Conn,
	name string,
	arguments interface{},
	msgID int64,
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
	msgObj["id"] = msgID

	// JSON encode
	msg, err := json.Marshal(msgObj)
	if err != nil {
		return err
	}

	w.log.Trace().Str("msg", string(msg)).Msg("ws send")

	if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
		var closeError websocket.CloseError
		if errors.As(err, &closeError) {
			if closeError.Code == websocket.StatusNormalClosure {
				w.log.Info().Msg("websocket closed cleanly")
				return io.EOF
			}
		} else if errors.Is(err, net.ErrClosed) {
			w.log.Info().Msg("websocket closed")
			return io.EOF
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
		w.msgID.Add(1)
	}()

	// Bump msgID
	msgID := w.msgID.Load()

	done := make(chan struct{}, 1)
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
	expectedID int64,
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

// FetchPlaylist fetches the playlist.
func (w *WebSocket) FetchPlaylist(
	ctx context.Context,
	conn *websocket.Conn,
	msgChan chan *WSResponse,
	expectedMode int,
) (playlist Playlist, availables []Playlist, err error) {
	hlsInfo, err := w.GetHLSInformation(ctx, conn, msgChan)
	if err != nil {
		return playlist, availables, err
	}

	playlists := SortPlaylists(ExtractAndMergePlaylists(hlsInfo))

	playlist, err = GetPlaylistOrBest(
		playlists,
		expectedMode,
	)
	if err != nil {
		return playlist, playlists, err
	}
	if expectedMode != playlist.Mode {
		return playlist, playlists, ErrQualityNotAvailable
	}

	return playlist, playlists, nil
}
