package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DataMetrics    *prometheus.CounterVec
	ServiceMetrics *prometheus.CounterVec
)

func init() {
	DataMetrics = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "datailed_data_access_metrics",
		Help: "The packages, methods and possible errors when accessing data",
	}, []string{"pkg", "method", "err"})

	ServiceMetrics = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "datailed_service_layer_metrics",
		Help: "TODO: ",
	}, []string{"function", "method", "sc", "err"})
}
