package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// JSONRequestLogger writes request metadata without request headers or bodies,
// preventing credentials and proof data from entering application logs.
func JSONRequestLogger(output io.Writer) fiber.Handler {
	if output == nil {
		output = os.Stdout
	}
	return func(c *fiber.Ctx) error {
		started := time.Now()
		err := c.Next()
		requestID, _ := c.Locals("requestid").(string)
		_ = json.NewEncoder(output).Encode(map[string]any{
			"timestamp":  started.UTC().Format(time.RFC3339Nano),
			"request_id": requestID,
			"method":     c.Method(),
			"path":       c.Path(),
			"status":     c.Response().StatusCode(),
			"latency_ms": time.Since(started).Milliseconds(),
		})
		return err
	}
}

type MetricsSnapshot struct {
	Requests       uint64            `json:"requests"`
	TotalLatencyMS int64             `json:"total_latency_ms"`
	Statuses       map[string]uint64 `json:"statuses"`
}

// RequestMetrics aggregates only operational counters; it never retains
// request headers, bodies, IP addresses, user IDs, or paths.
type RequestMetrics struct {
	mu       sync.Mutex
	requests uint64
	latency  int64
	statuses map[string]uint64
}

func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{statuses: make(map[string]uint64)}
}

var DefaultRequestMetrics = NewRequestMetrics()

func (m *RequestMetrics) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		started := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			fiberError := new(fiber.Error)
			if errors.As(err, &fiberError) {
				status = fiberError.Code
			}
		}
		m.mu.Lock()
		m.requests++
		m.latency += time.Since(started).Milliseconds()
		m.statuses[strconv.Itoa(status)]++
		m.mu.Unlock()
		return err
	}
}

func (m *RequestMetrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	statuses := make(map[string]uint64, len(m.statuses))
	for status, count := range m.statuses {
		statuses[status] = count
	}
	return MetricsSnapshot{Requests: m.requests, TotalLatencyMS: m.latency, Statuses: statuses}
}

func MetricsHandler(metrics *RequestMetrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(metrics.Snapshot())
	}
}
