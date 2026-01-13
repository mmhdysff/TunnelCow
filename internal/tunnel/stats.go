package tunnel

import (
	"io"
	"sync/atomic"
	"time"
)

type TrafficStats struct {
	BytesUp   uint64    `json:"bytes_up"`
	BytesDown uint64    `json:"bytes_down"`
	LatencyMs int64     `json:"latency_ms"`
	StartTime time.Time `json:"-"`
}

var GlobalStats = &TrafficStats{
	StartTime: time.Now(),
}

type MonitoredReader struct {
	R       io.Reader
	Counter *uint64
}

func (m *MonitoredReader) Read(p []byte) (n int, err error) {
	n, err = m.R.Read(p)
	if n > 0 {
		atomic.AddUint64(m.Counter, uint64(n))
	}
	return
}

type MonitoredWriter struct {
	W       io.Writer
	Counter *uint64
}

func (m *MonitoredWriter) Write(p []byte) (n int, err error) {
	n, err = m.W.Write(p)
	if n > 0 {
		atomic.AddUint64(m.Counter, uint64(n))
	}
	return
}
