package promtinytcp

import (
	"github.com/mkorman9/tinytcp"
	"github.com/prometheus/client_golang/prometheus"
)

// Config specifies an optional config for NewHandler.
type Config struct {
	// Namespace is a parameter attached to all Prometheus metrics registered in NewHandler.
	Namespace string

	// Subsystem is a parameter attached to all Prometheus metrics registered in NewHandler.
	Subsystem string
}

// NewHandler creates a metrics handler for tinytcp.Server. It can be registered using OnMetricsUpdate method.
// Created handler exposes all server metrics to the given prometheus.Registerer.
func NewHandler(
	registerer prometheus.Registerer,
	config ...*Config,
) func(metrics tinytcp.ServerMetrics) {
	c := &Config{}
	if config != nil {
		c = config[0]
	}

	totalRead := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "total_read",
		Help:      "Total number of bytes read by the server.",
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
	})
	totalWritten := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "total_written",
		Help:      "Total number of bytes written by the server.",
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
	})
	readLastSecond := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "read_last_second",
		Help:      "Total number of bytes read by the server last second.",
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
	})
	writtenLastSecond := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "written_last_second",
		Help:      "Total number of bytes written by the server last second.",
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
	})
	connections := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "connections",
		Help:      "Total number of active connections during the last second.",
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
	})
	goroutines := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "goroutines",
		Help:      "Total number of active goroutines during the last second.",
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
	})

	registerer.MustRegister(
		totalRead,
		totalWritten,
		readLastSecond,
		writtenLastSecond,
		connections,
		goroutines,
	)

	return func(metrics tinytcp.ServerMetrics) {
		totalRead.Set(float64(metrics.TotalRead))
		totalWritten.Set(float64(metrics.TotalWritten))
		readLastSecond.Set(float64(metrics.ReadLastSecond))
		writtenLastSecond.Set(float64(metrics.WrittenLastSecond))
		connections.Set(float64(metrics.Connections))
		goroutines.Set(float64(metrics.Goroutines))
	}
}
