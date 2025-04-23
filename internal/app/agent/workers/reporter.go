package workers

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/app/agent/metrics"
	httpretry "github.com/PiskarevSA/go-advanced/internal/app/agent/workers/http_retry"
	"github.com/PiskarevSA/go-advanced/internal/models"
)

type Reporter struct {
	wg            *sync.WaitGroup
	index         int
	metricsChan   <-chan metrics.Metrics
	serverAddress string
	key           string
	httpClient    *http.Client
}

func NewReporter(
	wg *sync.WaitGroup, index int, metricsChan <-chan metrics.Metrics,
	serverAddress string, key string,
) *Reporter {
	return &Reporter{
		wg:            wg,
		index:         index,
		metricsChan:   metricsChan,
		serverAddress: serverAddress,
		key:           key,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: httpretry.NewRetryableTransport(),
		},
	}
}

func (r *Reporter) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-ctx.Done():
				slog.Info("[reporter] stopping",
					"index", r.index,
					"reason", ctx.Err())
				return
			case metric, ok := <-r.metricsChan:
				if !ok {
					slog.Info("[reporter] stopping",
						"index", r.index,
						"reason", "metrics channel closed")
					return
				}
				if err := r.report(metric.Gauge, metric.Counter); err != nil {
					slog.Error("[reporter] report failed",
						"index", r.index,
						"error", err)
				} else {
					slog.Info("[reporter] report succeeded",
						"index", r.index)
				}
			}
		}
	}()
}

func (r *Reporter) report(gauge map[string]metrics.Gauge,
	counter map[string]metrics.Counter,
) error {
	url := "http://" + r.serverAddress + "/updates/"

	metrics := make([]models.Metric, 0, len(gauge)+len(counter))
	for key, gauge := range gauge {
		value := float64(gauge)
		m := models.Metric{
			ID:    key,
			MType: "gauge",
			Value: &value,
		}
		metrics = append(metrics, m)
	}

	for key, counter := range counter {
		delta := int64(counter)
		m := models.Metric{
			ID:    key,
			MType: "counter",
			Delta: &delta,
		}
		metrics = append(metrics, m)
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	if err := r.reportToURL(url, body, r.key); err != nil {
		return err
	}
	return nil
}

func (r *Reporter) reportToURL(url string, body []byte, key string) error {
	compressedBodyBuffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(compressedBodyBuffer)
	// write compressed body to buffer
	if _, err := gzipWriter.Write(body); err != nil {
		return fmt.Errorf("gzipWriter.Write(): %w", err)
	}
	// flush any unwritten data to buffer
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("gzipWriter.Close(): %w", err)
	}

	var hexSum string
	if len(key) > 0 {
		h := hmac.New(sha256.New, []byte(key))
		compressedBody := compressedBodyBuffer.Bytes()
		h.Write(compressedBody)
		sign := h.Sum(nil)
		hexSum = hex.EncodeToString(sign[:])
	}

	req, err := http.NewRequest(http.MethodPost, url, compressedBodyBuffer)
	if err != nil {
		return fmt.Errorf("http.NewRequest(): %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if len(hexSum) > 0 {
		req.Header.Set("HashSHA256", hexSum)
	}
	res, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("httpClient.Do(): %w", err)
	}
	defer res.Body.Close()

	// The default HTTP client's Transport may not
	// reuse HTTP/1.x "keep-alive" TCP connections if the Body is
	// not read to completion and closed.
	_, err = io.Copy(io.Discard, res.Body)
	if err != nil {
		return fmt.Errorf("io.Copy(): %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %v returns %v", url, res.Status)
	}

	return nil
}
