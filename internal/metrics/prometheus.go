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
		Namespace:   "cffc",
		Subsystem:   "userservice",
		Name:        "database",
		Help:        "The packages, methods and possible errors when accessing data",
		ConstLabels: prometheus.Labels{},
	}, []string{"db", "pkg", "function", "status"})

	ServiceMetrics = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "cffc",
		Subsystem:   "userservice",
		Name:        "router",
		Help:        "Service requests tracked by ???",
		ConstLabels: prometheus.Labels{},
	}, []string{"url", "proto", "method", "sc"})
}

func NewHandler() http.HandlerFunc {
	reg := prometheus.NewRegistry()

	reg.MustRegister(DataMetrics)
	reg.MustRegister(ServiceMetrics)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}).ServeHTTP
}
