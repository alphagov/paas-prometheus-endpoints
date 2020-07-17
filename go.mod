module github.com/alphagov/paas-prometheus-endpoints

go 1.13

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/alphagov/paas-elasticache-broker v0.20.0
	github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator v0.0.0-00010101000000-000000000000
	github.com/alphagov/paas-prometheus-endpoints/pkg/config v0.0.0-00010101000000-000000000000
	github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint v0.0.0-00010101000000-000000000000
	github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher v0.0.0-00010101000000-000000000000
	github.com/aws/aws-sdk-go v1.33.6
	github.com/cloudfoundry-community/go-cfclient v0.0.0-20200413172050-18981bf12b4b
	github.com/gin-gonic/gin v1.6.3
	github.com/gojektech/heimdall v5.0.2+incompatible // indirect
	github.com/gojektech/valkyrie v0.0.0-20190210220504-8f62c1e7ba45 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a // indirect
	github.com/lib/pq v0.0.0-20180327071824-d34b9ff171c2 // indirect
	github.com/prometheus/client_golang v1.2.1
	github.com/satori/go.uuid v1.2.0 // indirect
)

// Miki couldn't get the imports to work without this but doesn't fully understand
// https://www.reddit.com/r/golang/comments/ah0w1q/modules_and_local_imports/

replace github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator => ./pkg/authenticator

replace github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher => ./pkg/service_plans_fetcher

replace github.com/alphagov/paas-prometheus-endpoints/pkg/config => ./pkg/config

replace github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint => ./pkg/metric_endpoint
