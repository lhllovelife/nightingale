package cstats

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const Service = "n9e-center" // 定义了一个常量 Service，表示服务的名称

var (
	// 定义了一组标签，这些标签用于给 Prometheus 指标添加维度，以便更细粒度地监控数据
	labels = []string{"service", "code", "path", "method"}

	/*
		1. uptime 是一个 CounterVec，用于记录服务的运行时间。CounterVec 是一个计数器向量，可以根据标签进行区分(Counter类型，单调递增)
		# HELP uptime HTTP service uptime.
		# TYPE uptime counter
		uptime{service="n9e-center"} 12345

		查看服务的运行时间: uptime{service="n9e-center"}
	*/
	uptime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "uptime",
			Help: "HTTP service uptime.",
		}, []string{"service"},
	)

	/*
		2. RequestCounter 是一个 CounterVec，用于记录 HTTP 请求的总数。(Counter类型，单调递增)
		# HELP http_request_count_total Total number of HTTP requests made.
		# TYPE http_request_count_total counter
		http_request_count_total{service="n9e-center",code="200",path="/",method="GET"} 678

		查看 HTTP 请求的总数: http_request_count_total{service="n9e-center"}
	*/
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_count_total",
			Help: "Total number of HTTP requests made.",
		}, labels,
	)

	/*
		3. RequestDuration 是一个 HistogramVec，用于记录 HTTP 请求的持续时间，使用定义的桶（Buckets）来记录不同范围的持续时间。
		Histogram（直方图类型）用于描述数据分布

		# HELP http_request_duration_seconds HTTP request latencies in seconds.
		# TYPE http_request_duration_seconds histogram
		http_request_duration_seconds_bucket{service="n9e-center",code="200",path="/",method="GET",le="0.01"} 10
		http_request_duration_seconds_bucket{service="n9e-center",code="200",path="/",method="GET",le="0.1"} 50
		http_request_duration_seconds_bucket{service="n9e-center",code="200",path="/",method="GET",le="1"} 100
		http_request_duration_seconds_bucket{service="n9e-center",code="200",path="/",method="GET",le="10"} 110
		http_request_duration_seconds_sum{service="n9e-center",code="200",path="/",method="GET"} 5.5
		http_request_duration_seconds_count{service="n9e-center",code="200",path="/",method="GET"} 110

		10 个请求的延迟小于等于 0.01 秒。
		50 个请求的延迟小于等于 0.1 秒。
		100 个请求的延迟小于等于 1 秒。
		110 个请求的延迟小于等于 10 秒

	*/
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Buckets: []float64{.01, .1, 1, 10},
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds.",
		}, labels,
	)
)

// 3. 查看 HTTP 请求的持续时间分布

func Init() {
	// Metrics 数据指标必须被注册后才会被公开
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(
		uptime,
		RequestCounter,
		RequestDuration,
	)

	go recordUptime()
}

// recordUptime increases service uptime per second.
func recordUptime() {
	for range time.Tick(time.Second) {
		uptime.WithLabelValues(Service).Inc()
	}
}
