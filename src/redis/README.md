# `paas-prometheus-endpoints-redis`

This provides a Prometheus metrics endpoint for GOV.UK PaaS tenants to obtain metrics about their Redis services. The metric values cover the last 5 minutes (300 seconds.) For cost reasons it is quite important to not scrape it more regularly than every 5 minutes.

Our Redis service is provided by AWS ElastiCache Redis, and automated by our [paas-elasticache-broker](https://github.com/alphagov/paas-elasticache-broker). This codebase exports Redis metrics from AWS CloudWatch Metrics, which ElastiCache automatically feeds Redis metrics into.

This is available to any PaaS tenant who wants it, but it costs money to use and we can’t recharge that until we make billing improvements. For now it's available on request.

## Metrics exported

From [ElastiCache Metrics for Redis](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.Redis.html):

* `curr_items`
* `cache_hit_rate`
* `evictions`
* `curr_connections`
* `new_connections`
* `database_memory_usage_percentage`

From [ElastiCache Host-Level Metrics](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheMetrics.HostLevel.html):

* `cpu_utilization`
* `swap_usage`
* `network_bytes_in`
* `network_bytes_out`

We export three statistics about each metric. Each has a `_avg`, `_max` and `_min` value (for example `cpu_utilization_max`.) These values cover a 5-minute window.

Many more metrics are available than are currently exported. However getting more values from CloudWatch Metrics would cost more money. Future work could fetch most metrics directly from the Redis nodes to avoid AWS API charges.

## Cost

The cost comes from how it gets the metrics. It makes API calls to AWS CloudFront Metrics. The numbers get quite deep but the TL;DR is that each call to the metrics endpoint costs $0.0001 for each non-HA Redis service and $0.0002 for every HA Redis service. Scraping every 5 minutes, there'll be about 8640 scrapes/month, meaning it's $1-2/month per Redis service.

We won't be routinely recharging these costs yet. In time when we have improved our billing system we might. We reserve the right to recoup vast costs incurred by misusing the endpoint but we're happy to accept the cost if this is used as described.

## Time granularity

You get data covering the last five minutes. To minimise costs it's quite important that you **scrape every five minutes**. You get `_avg`, `_min` and `_max` values of each of the metrics, so you don't lose too much information.

Prometheus doesn't like importing bulk historical data, but that's the cheapest way to export from CloudWatch Metrics. Polling CloudWatch Metrics is expensive but we found that polling every 5 minutes has pretty acceptable costs even for large PaaS users.

## Setup

1. Create a new PaaS user
1. Give the new user Space Auditor permissions on every space with Redises you want metrics for (the Org Auditor permission doesn't work for this)
1. Configure your Prometheus to scrape `https://redis.metrics.[london.]cloud.service.gov.uk/metrics`:
  
    * Provide the PaaS user's username and password to Prometheus for use as basic auth credentials
    * Set the scrape period to 5 minutes (300 seconds)
1. Within a few minutes you should now have metrics coming into your Prometheus

We strongly suggest not giving that PaaS user any write permissions, only auditor permissions. This ensures that someone who breaks into Prometheus can't start modifying or accessing your resources in PaaS.

Here is an example Prometheus config, which will rename the metrics to `paas_redis_*` be more easily discoverable:

```yaml
scrape_configs:
- job_name: paas_redis_metrics
  scheme: https
  basic_auth:
    username: USERNAME_OF_THE_AUDITOR_USER_YOU_CREATED
    password: PASSWORD_OF_THE_AUDITOR_USER_YOU_CREATED
  static_configs:
  - targets:
    - redis.metrics.london.cloud.service.gov.uk
  metrics_path: /metrics
  scrape_interval: 300s
  scrape_timeout: 120s
  honor_timestamps: true
  metric_relabel_configs:
  # Prepend `paas_redis_` so the metrics are easier to find
  - action: replace
    source_labels: [__name__]
    target_label: __name__
    regex: (.*)
    replacement: paas_redis_${1}
```
