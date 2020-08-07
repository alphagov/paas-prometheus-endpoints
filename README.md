# `paas-prometheus-endpoints`

A collection of Prometheus endpoints used by GOV.UK PaaS tenants to collect metrics about their databases.

## Redis

The Redis endpoint exports metrics about AWS ElastiCache Redis services automated by the [paas-elasticache-broker](https://github.com/alphagov/paas-elasticache-broker). It fetches metrics from AWS CloudWatch Metrics, which costs money to do.

Further information is available in `src/redis/README.md`.
