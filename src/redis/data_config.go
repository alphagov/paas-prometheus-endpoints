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
	"CacheHitRate":                  "cache_hit_rate",
	"Evictions":                     "evictions",
	"CurrConnections":               "curr_connections",
	"NewConnections":                "new_connections",
	"DatabaseMemoryUsagePercentage": "database_memory_usage_percentage",

	"CPUUtilization":  "cpu_utilization",
	"SwapUsage":       "swap_usage",
	"NetworkBytesIn":  "network_bytes_in",
	"NetworkBytesOut": "network_bytes_out",
}

// https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.Redis.html
var cacheClusterMetrics = []string{
	"CurrItems",
	"CacheHitRate",
	"Evictions",
	"CurrConnections",
	"NewConnections",
	"DatabaseMemoryUsagePercentage",
}

// https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.HostLevel.html
var hostMetrics = []string{
	"CPUUtilization",
	"SwapUsage",
	"NetworkBytesIn",
	"NetworkBytesOut",
}
