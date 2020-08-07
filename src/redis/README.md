# `paas-prometheus-endpoints-redis`

This provides a Prometheus metrics endpoint for GOV.UK PaaS tenants to obtain metrics about their Redis services.

Our Redis service is provided by AWS ElastiCache Redis, and automated by our [paas-elasticache-broker](https://github.com/alphagov/paas-elasticache-broker). This codebase exports Redis metrics from AWS CloudWatch Metrics, which ElastiCache automatically feeds Redis metrics into.

## Metrics exported

We export the following metrics with `_average`, `_maximum` and `_minimum` appended:

* `curr_items`
* `curr_connections`
* `new_connections`
* `cpu_utilization`
* `engine_cpu_utilization`
* `database_memory_usage_percentage`
* `swap_usage`

These values are described in more detail at https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.Redis.html and https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.HostLevel.html.

Many more metrics are available than are currently exported. Future work could fetch most metrics directly from the Redis nodes to avoid AWS API charges, allowing more metrics to be exported.

## Constraints

AWS CloudWatch Metrics charges for API calls. At the time of writing it costs $0.01 per 1000 metric data entries. Each metric data entries can be one value, or a list of one value per minute over an hour. Unfortunately we can't take advantage of this to bulk-download metrics data because Prometheus dislikes old data.

Here's an example of how the costs add up:

* Assume we will export the 10 most useful metrics, rather than the dozens available
* CloudWatch Metrics stores one value per minute for average, minimum and maximum of each metric. We can just about fetch these statistics at no additional cost.
* We'll fetch one value per minute and tell tenants to have their Prometheus scrape the endpoint every 60 seconds
* Say we have a tenant with 100 Redis instances (this is realistic at the time of writing)
* The cost of providing metrics to that tenant is: `number_of_minutes_per_month * number_of_metrics * number_of_redis_instances * cost_per_metric`
* This works out as `43200 * 10 * 100 * 0.01 / 1000 = $432 per month`

We don't want to pay that much. Our current compromise is to only provide the most actionable metrics, and to fetch values over five minutes (as opposed to the one-minute resolution in CloudWatch Metrics.) This is mitigated a little by fetching the average, minimum and maximum values.

CloudWatch Metrics offers many more Redis metrics than are exposed here, but fetching them would cost more money. It is possible to obtain a lot of those metrics from the Redis nodes themselves, and that would make this exporter cheaper. Useful future work.
