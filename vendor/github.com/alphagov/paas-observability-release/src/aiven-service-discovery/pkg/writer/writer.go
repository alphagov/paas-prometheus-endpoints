package writer

import (
	"encoding/json"
	"io/ioutil"

	"code.cloudfoundry.org/lager"
)

func init() {
	initMetrics()
}

type Writer interface {
	WritePrometheusTargetConfigs(targets []PrometheusTargetConfig)
}

type writer struct {
	filePath string

	logger lager.Logger
}

func NewWriter(
	filePath string,
	logger lager.Logger,
) Writer {
	lsession := logger.Session("writer", lager.Data{"file-path": filePath})

	return &writer{
		filePath: filePath,

		logger: lsession,
	}
}

func (w *writer) WritePrometheusTargetConfigs(targets []PrometheusTargetConfig) {
	lsession := w.logger.Session("write-prometheus-target-configs")
	lsession.Info("begin")
	defer lsession.Info("end")

	WriterWriteTargetsTotal.Inc()

	targetsAsJSON, err := json.Marshal(targets)

	if err != nil {
		lsession.Error(
			"err-marshal-json-targets",
			err, lager.Data{"targets": targets},
		)

		WriterWriteTargetsErrorsTotal.Inc()

		return
	}

	err = ioutil.WriteFile(w.filePath, targetsAsJSON, 0644)
	if err != nil {
		lsession.Error("err-write-json-targets", err)

		WriterWriteTargetsErrorsTotal.Inc()

		return
	}
}
