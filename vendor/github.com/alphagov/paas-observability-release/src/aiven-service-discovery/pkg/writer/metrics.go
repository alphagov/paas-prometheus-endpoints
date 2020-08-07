package writer

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WriterWriteTargetsErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "writer_write_targets_errors_total",
		Help: "Counter of total number of target file write failures",
	})

	WriterWriteTargetsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "writer_write_targets_total",
		Help: "Counter of total number of target file writes",
	})
)

func initMetrics() {
	prometheus.MustRegister(WriterWriteTargetsErrorsTotal)
	prometheus.MustRegister(WriterWriteTargetsTotal)
}
