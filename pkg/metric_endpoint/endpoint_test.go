package metric_endpoint_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/alphagov/paas-prometheus-endpoints/pkg/authenticator"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/metric_endpoint"
	"github.com/alphagov/paas-prometheus-endpoints/pkg/service_plans_fetcher"

	"code.cloudfoundry.org/lager"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
)

type MockServicePlansStore struct {
	MockService      *cfclient.Service
	MockServicePlans []cfclient.ServicePlan
}

func (m *MockServicePlansStore) GetService() *cfclient.Service {
	return m.MockService
}

func (m *MockServicePlansStore) GetServicePlans() []cfclient.ServicePlan {
	return m.MockServicePlans
}

var _ service_plans_fetcher.ServicePlansStore = (*MockServicePlansStore)(nil)

type MockMetricFetcher struct {
	FetchMetricsCallback func(
		_ *gin.Context,
		_ authenticator.User,
		_ []cfclient.ServiceInstance,
		_ []cfclient.ServicePlan,
		_ cfclient.Service,
	) (metric_endpoint.Metrics, error)
}

func (f *MockMetricFetcher) FetchMetrics(
	c *gin.Context,
	user authenticator.User,
	serviceInstances []cfclient.ServiceInstance,
	servicePlans []cfclient.ServicePlan,
	service cfclient.Service,
) (metric_endpoint.Metrics, error) {
	return f.FetchMetricsCallback(c, user, serviceInstances, servicePlans, service)
}

var _ = Describe("Metric Endpoint", func() {
	var logger lager.Logger
	var router *gin.Engine
	var mockUser *authenticator.MockUser
	var mockServicePlansStore *MockServicePlansStore
	var mockMetricFetcher *MockMetricFetcher

	BeforeEach(func() {
		logger = lager.NewLogger("metric-endpoint-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

		mockServicePlansStore = &MockServicePlansStore{}
		mockUser = &authenticator.MockUser{
			MockUsername: "mock-user",
		}
		mockMetricFetcher = &MockMetricFetcher{}

		router = gin.Default()
		router.Use(func(c *gin.Context) {
			c.Set("authenticated_user", mockUser)
			c.Next()
		})
		router.GET("/metrics", metric_endpoint.MetricEndpoint(mockServicePlansStore, mockMetricFetcher, logger))
	})

	It("errors if it doesn't know what CF service to get metrics for", func() {
		mockServicePlansStore.MockService = nil

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)
		Expect(w.Body.String()).To(MatchJSON(`{"message": "an error occurred when trying to fetch the service"}`))
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})

	It("fetches metrics for the user's service instances of the right CF service", func() {
		mockServicePlansStore.MockService = &cfclient.Service{Guid: "fake-service-1-guid"}
		mockServicePlansStore.MockServicePlans = []cfclient.ServicePlan{
			{Guid: "fake-service-1-plan-1-guid"},
			{Guid: "fake-service-1-plan-2-guid"},
		}
		mockUser.MockServiceInstances = []cfclient.ServiceInstance{
			{
				Name:            "service-instance-1",
				ServicePlanGuid: "fake-service-1-plan-1-guid",
			},
			{
				Name:            "service-instance-2",
				ServicePlanGuid: "fake-service-1-plan-2-guid",
			},
			{
				Name:            "service-instance-2",
				ServicePlanGuid: "fake-service-2-plan-1-guid",
			},
		}

		mockMetricFetcher.FetchMetricsCallback = func(
			_ *gin.Context,
			user authenticator.User,
			serviceInstances []cfclient.ServiceInstance,
			servicePlans []cfclient.ServicePlan,
			service cfclient.Service,
		) (metric_endpoint.Metrics, error) {
			Expect(service).To(Equal(*mockServicePlansStore.MockService))
			Expect(servicePlans).To(Equal(mockServicePlansStore.MockServicePlans))
			Expect(serviceInstances).To(ConsistOf(
				mockUser.MockServiceInstances[0],
				mockUser.MockServiceInstances[1],
			))
			Expect(user).To(Equal(mockUser))
			return nil, nil
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("renders metrics in Prometheus format", func() {
		mockServicePlansStore.MockService = &cfclient.Service{Guid: "fake-service-1-guid"}
		mockServicePlansStore.MockServicePlans = []cfclient.ServicePlan{
			{Guid: "fake-service-1-plan-1-guid"},
			{Guid: "fake-service-1-plan-2-guid"},
		}
		mockUser.MockServiceInstances = []cfclient.ServiceInstance{
			{
				Name:            "service-instance-1",
				ServicePlanGuid: "fake-service-1-plan-1-guid",
			},
			{
				Name:            "service-instance-2",
				ServicePlanGuid: "fake-service-1-plan-2-guid",
			},
			{
				Name:            "service-instance-3",
				ServicePlanGuid: "fake-service-2-plan-1-guid",
			},
		}

		mockMetricFetcher.FetchMetricsCallback = func(
			_ *gin.Context,
			user authenticator.User,
			serviceInstances []cfclient.ServiceInstance,
			servicePlans []cfclient.ServicePlan,
			service cfclient.Service,
		) (metric_endpoint.Metrics, error) {
			serviceInstanceName := "service_instance_name"
			metrics := []*dto.Metric{}
			for i, serviceInstance := range serviceInstances {
				fi := float64(i)
				name := serviceInstance.Name
				metrics = append(metrics, &dto.Metric{
					Gauge: &dto.Gauge{Value: &fi},
					Label: []*dto.LabelPair{
						{
							Name:  &serviceInstanceName,
							Value: &name,
						},
					},
				})
			}
			serviceInstanceIndex := "service_instance_index"
			gauge := dto.MetricType_GAUGE
			return metric_endpoint.Metrics{
				"service_instance_index": &dto.MetricFamily{
					Name:   &serviceInstanceIndex,
					Type:   &gauge,
					Metric: metrics,
				},
			}, nil
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal(`# TYPE service_instance_index gauge
service_instance_index{service_instance_name="service-instance-1"} 0
service_instance_index{service_instance_name="service-instance-2"} 1
`))
	})
})
