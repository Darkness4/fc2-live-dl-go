package logger

import (
	"net/http"

	"go.uber.org/zap"
)

type Transport struct {
	Transport http.RoundTripper
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	I.Info("http req",
		zap.String("req.method", req.Method),
		zap.String("req.url", req.URL.String()),
		zap.Any("req.headers", req.Header),
	)

	// Call the underlying RoundTripper to make the actual request
	res, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	I.Info("http resp",
		zap.String("req.method", req.Method),
		zap.String("req.url", req.URL.String()),
		zap.Any("req.headers", req.Header),
		zap.Any("res.headers", res.Header),
		zap.Any("res.status", res.StatusCode),
	)

	return res, nil
}
