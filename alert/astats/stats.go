package astats

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "n9e"
	subsystem = "alert"
)

type Stats struct {
	AlertNotifyTotal            *prometheus.CounterVec
	AlertNotifyErrorTotal       *prometheus.CounterVec
	CounterAlertsTotal          *prometheus.CounterVec
	GaugeAlertQueueSize         prometheus.Gauge
	CounterRuleEval             *prometheus.CounterVec
	CounterQueryDataErrorTotal  *prometheus.CounterVec
	CounterQueryDataTotal       *prometheus.CounterVec
	CounterRecordEval           *prometheus.CounterVec
	CounterRecordEvalErrorTotal *prometheus.CounterVec
	CounterMuteTotal            *prometheus.CounterVec
	CounterRuleEvalErrorTotal   *prometheus.CounterVec
	CounterHeartbeatErrorTotal  *prometheus.CounterVec
	CounterSubEventTotal        *prometheus.CounterVec
}

func NewSyncStats() *Stats {
	// 1. 规则评估的次数
	CounterRuleEval := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "rule_eval_total",
		Help:      "Number of rule eval.",
	}, []string{})

	// 2. 规则评估错误的次数
	CounterRuleEvalErrorTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "rule_eval_error_total",
		Help:      "Number of rule eval error.",
	}, []string{"datasource", "stage"})

	// 3. 查询数据错误的次数
	CounterQueryDataErrorTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "query_data_error_total",
		Help:      "Number of rule eval query data error.",
	}, []string{"datasource"})

	// 4. 查询数据的次数
	CounterQueryDataTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "query_data_total",
		Help:      "Number of rule eval query data.",
	}, []string{"datasource"})

	// 5. 记录评估总数
	CounterRecordEval := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "record_eval_total",
		Help:      "Number of record eval.",
	}, []string{"datasource"})

	// 6. 记录评估错误总数
	CounterRecordEvalErrorTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "record_eval_error_total",
		Help:      "Number of record eval error.",
	}, []string{"datasource"})

	// 7. 发送通知消息总数
	AlertNotifyTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "alert_notify_total",
		Help:      "Number of send msg.",
	}, []string{"channel"})

	// 8. 发送通知消息错误总数
	AlertNotifyErrorTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "alert_notify_error_total",
		Help:      "Number of send msg.",
	}, []string{"channel"})

	// 9. 产生的告警总量
	CounterAlertsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "alerts_total",
		Help:      "Total number alert events.",
	}, []string{"cluster", "type", "busi_group"})

	// 10. 内存中的告警事件队列的长度
	GaugeAlertQueueSize := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "alert_queue_size",
		Help:      "The size of alert queue.",
	})

	// 11.  告警抑制数量
	CounterMuteTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "mute_total",
		Help:      "Number of mute.",
	}, []string{"group"})

	// 12. 子事件总数
	CounterSubEventTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "sub_event_total",
		Help:      "Number of sub event.",
	}, []string{"group"})

	// 13. 心跳处理失败数量
	CounterHeartbeatErrorTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "heartbeat_error_count",
		Help:      "Number of heartbeat error.",
	}, []string{})

	prometheus.MustRegister(
		CounterAlertsTotal,
		GaugeAlertQueueSize,
		AlertNotifyTotal,
		AlertNotifyErrorTotal,
		CounterRuleEval,
		CounterQueryDataTotal,
		CounterQueryDataErrorTotal,
		CounterRecordEval,
		CounterRecordEvalErrorTotal,
		CounterMuteTotal,
		CounterRuleEvalErrorTotal,
		CounterHeartbeatErrorTotal,
		CounterSubEventTotal,
	)

	return &Stats{
		CounterAlertsTotal:          CounterAlertsTotal,
		GaugeAlertQueueSize:         GaugeAlertQueueSize,
		AlertNotifyTotal:            AlertNotifyTotal,
		AlertNotifyErrorTotal:       AlertNotifyErrorTotal,
		CounterRuleEval:             CounterRuleEval,
		CounterQueryDataTotal:       CounterQueryDataTotal,
		CounterQueryDataErrorTotal:  CounterQueryDataErrorTotal,
		CounterRecordEval:           CounterRecordEval,
		CounterRecordEvalErrorTotal: CounterRecordEvalErrorTotal,
		CounterMuteTotal:            CounterMuteTotal,
		CounterRuleEvalErrorTotal:   CounterRuleEvalErrorTotal,
		CounterHeartbeatErrorTotal:  CounterHeartbeatErrorTotal,
		CounterSubEventTotal:        CounterSubEventTotal,
	}
}
