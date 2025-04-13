package agent

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

const retryCount = 3

type retryableTransport struct {
	transport http.RoundTripper
}

func (t *retryableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		if bodyBytes, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	res, err := t.transport.RoundTrip(req)
	retries := 0
	for shouldRetry(err, res) && retries < retryCount {
		time.Sleep(backoff(retries))
		drainBody(res)
		if req.Body != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		res, err = t.transport.RoundTrip(req)
		retries++
	}
	return res, err
}

func shouldRetry(err error, res *http.Response) bool {
	return err != nil ||
		res.StatusCode == http.StatusBadGateway ||
		res.StatusCode == http.StatusServiceUnavailable ||
		res.StatusCode == http.StatusGatewayTimeout
}

func backoff(retries int) time.Duration {
	return time.Duration(1+2*retries) * time.Second
}

func drainBody(res *http.Response) {
	if res != nil && res.Body != nil {
		_, _ = io.Copy(io.Discard, res.Body)
		res.Body.Close()
	}
}
