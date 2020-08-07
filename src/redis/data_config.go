package main

// Maps from CloudWatch names to Prometheus-alike names
// Ideally would be automated but handling acronyms in code was ugly

var statistics = map[string]string{
	"Average": "avg",
	"Minimum": "min",
	"Maximum": "max",
}

var metrics = map[string]string{
	"CurrItems":                     "curr_items",
	"CurrConnections":               "curr_connections",
	"NewConnections":                "new_connections",
	"CPUUtilization":                "cpu_utilization",
	"EngineCPUUtilization":          "engine_cpu_utilization",
	"DatabaseMemoryUsagePercentage": "database_memory_usage_percentage",
	"SwapUsage":                     "swap_usage",
}

// Metrics grouped by some boring logic around what "dimensions" to ask
// CloudFront Metrics for

var cacheClusterMetrics = []string{
	"CurrItems",
	"CurrConnections",
	"NewConnections",
	"EngineCPUUtilization",
	"DatabaseMemoryUsagePercentage",
}

var hostMetrics = []string{
	"CPUUtilization",
	"SwapUsage",
}
