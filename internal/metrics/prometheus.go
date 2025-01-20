package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DataMetrics     *prometheus.CounterVec
	ServiceMetrics3 *prometheus.CounterVec
	ServiceMetrics  *prometheus.CounterVec
)

func init() {
	DataMetrics = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "database",
		Help:      "The packages, methods and possible errors when accessing data",
		Namespace: "cffc",
		Subsystem: "userservice",
	}, []string{"db", "function", "err"})

	ServiceMetrics = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "router",
		Help:        "Service requests tracked by ???",
		ConstLabels: prometheus.Labels{},
		Namespace:   "cffc",
		Subsystem:   "userservice",
	}, []string{"url", "proto", "method", "sc"})
}

func NewHandler(reg *prometheus.Registry) http.HandlerFunc {
	reg.MustRegister(DataMetrics)
	reg.MustRegister(ServiceMetrics)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}).ServeHTTP
}
