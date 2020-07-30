package writer

import (
	"net"
)

type PrometheusTargetConfigLabels struct {
	ServiceName string `json:"aiven_service_name"`
	ServiceType string `json:"aiven_service_type"`
	Hostname    string `json:"aiven_hostname"`
	Plan        string `json:"aiven_plan"`
	Cloud       string `json:"aiven_cloud"`
	NodeCount   string `json:"aiven_node_count"`
}

type PrometheusTargetConfig struct {
	Targets []net.IP                     `json:"targets"`
	Labels  PrometheusTargetConfigLabels `json:"labels"`
}
