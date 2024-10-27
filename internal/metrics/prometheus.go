package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DataMetrics    *prometheus.CounterVec
	ServiceMetrics *prometheus.CounterVec
)

func init() {
	DataMetrics = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "datailed_data_access_metrics",
		Help: "The packages, methods and possible errors when accessing data",
	}, []string{"pkg", "app", "db", "function", "err"})

	ServiceMetrics = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "datailed_service_layer_metrics",
		Help: "Service requests tracked by method, function, statuscode, error",
	}, []string{"pkg", "app", "method", "function", "sc", "err"})
}

func NewHandler(reg *prometheus.Registry) http.HandlerFunc {
	reg.MustRegister(DataMetrics)
	reg.MustRegister(ServiceMetrics)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}).ServeHTTP
}
