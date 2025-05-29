// Package metrics 提供缓存运行时指标采集、统计和输出功能
package metrics

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	// 默认的Prometheus指标前缀
	defaultMetricPrefix = "hcache"
)

// PrometheusExporter 提供将缓存指标导出为Prometheus格式的功能
type PrometheusExporter struct {
	// 指标收集器引用
	metrics *Metrics

	// 指标前缀
	prefix string

	// 缓存名称，用于标签
	cacheName string

	// 上次导出时间
	lastExportTime time.Time

	// 互斥锁
	mu sync.Mutex
}

// NewPrometheusExporter 创建一个新的Prometheus导出器
func NewPrometheusExporter(metrics *Metrics, cacheName string) *PrometheusExporter {
	return &PrometheusExporter{
		metrics:        metrics,
		prefix:         defaultMetricPrefix,
		cacheName:      cacheName,
		lastExportTime: time.Now(),
	}
}

// SetPrefix 设置指标前缀
func (p *PrometheusExporter) SetPrefix(prefix string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prefix = prefix
}

// Export 导出Prometheus格式的指标
func (p *PrometheusExporter) Export() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 获取指标快照
	snapshot := p.metrics.GetSnapshot()
	if snapshot == nil {
		return ""
	}

	var buf bytes.Buffer
	p.lastExportTime = time.Now()

	// 添加基本指标
	p.addCounter(&buf, "hits_total", "Total number of cache hits", snapshot.Hits)
	p.addCounter(&buf, "misses_total", "Total number of cache misses", snapshot.Misses)
	p.addGauge(&buf, "hit_ratio", "Cache hit ratio", snapshot.HitRatio)

	p.addCounter(&buf, "sets_total", "Total number of cache sets", snapshot.Sets)
	p.addCounter(&buf, "updates_total", "Total number of cache updates", snapshot.Updates)
	p.addCounter(&buf, "overwrites_total", "Total number of cache overwrites", snapshot.Overwrites)
	p.addCounter(&buf, "rejects_total", "Total number of cache rejects", snapshot.Rejects)

	p.addCounter(&buf, "evictions_total", "Total number of cache evictions", snapshot.Evictions)
	p.addCounter(&buf, "expired_total", "Total number of expired cache items", snapshot.Expired)
	p.addCounter(&buf, "manually_deleted_total", "Total number of manually deleted cache items", snapshot.ManuallyDeleted)

	p.addGauge(&buf, "get_latency_ns", "Average get latency in nanoseconds", float64(snapshot.GetLatencyAvg))
	p.addGauge(&buf, "set_latency_ns", "Average set latency in nanoseconds", float64(snapshot.SetLatencyAvg))
	p.addGauge(&buf, "delete_latency_ns", "Average delete latency in nanoseconds", float64(snapshot.DeleteLatencyAvg))

	p.addGauge(&buf, "entry_count", "Number of entries in the cache", float64(snapshot.EntryCount))
	p.addGauge(&buf, "memory_usage_bytes", "Memory usage in bytes", float64(snapshot.MemoryUsage))

	// 添加分片指标
	if snapshot.ShardMetrics != nil {
		for _, sm := range snapshot.ShardMetrics {
			labels := fmt.Sprintf(`cache="%s",shard="%d"`, p.cacheName, sm.ShardID)
			p.addGaugeWithLabels(&buf, "shard_items", "Number of items in the shard", float64(sm.ItemCount), labels)
			p.addGaugeWithLabels(&buf, "shard_memory_bytes", "Memory usage of the shard in bytes", float64(sm.MemoryUsage), labels)
			p.addCounterWithLabels(&buf, "shard_hits_total", "Total number of hits in the shard", sm.Hits, labels)
			p.addCounterWithLabels(&buf, "shard_misses_total", "Total number of misses in the shard", sm.Misses, labels)
			p.addCounterWithLabels(&buf, "shard_evictions_total", "Total number of evictions in the shard", sm.Evictions, labels)
			p.addCounterWithLabels(&buf, "shard_conflicts_total", "Total number of conflicts in the shard", sm.Conflicts, labels)
		}
	}

	// 添加直方图数据
	if snapshot.LatencyHistogram != nil {
		p.addHistogram(&buf, "latency_histogram", "Latency histogram in nanoseconds", snapshot.LatencyHistogram)
	}

	return buf.String()
}

// addCounter 添加计数器类型指标
func (p *PrometheusExporter) addCounter(buf *bytes.Buffer, name, help string, value uint64) {
	metricName := fmt.Sprintf("%s_%s", p.prefix, name)
	fmt.Fprintf(buf, "# HELP %s %s\n", metricName, help)
	fmt.Fprintf(buf, "# TYPE %s counter\n", metricName)
	fmt.Fprintf(buf, "%s{cache=\"%s\"} %d\n\n", metricName, p.cacheName, value)
}

// addCounterWithLabels 添加带标签的计数器类型指标
func (p *PrometheusExporter) addCounterWithLabels(buf *bytes.Buffer, name, help string, value uint64, labels string) {
	metricName := fmt.Sprintf("%s_%s", p.prefix, name)
	fmt.Fprintf(buf, "# HELP %s %s\n", metricName, help)
	fmt.Fprintf(buf, "# TYPE %s counter\n", metricName)
	fmt.Fprintf(buf, "%s{%s} %d\n\n", metricName, labels, value)
}

// addGauge 添加仪表类型指标
func (p *PrometheusExporter) addGauge(buf *bytes.Buffer, name, help string, value float64) {
	metricName := fmt.Sprintf("%s_%s", p.prefix, name)
	fmt.Fprintf(buf, "# HELP %s %s\n", metricName, help)
	fmt.Fprintf(buf, "# TYPE %s gauge\n", metricName)
	fmt.Fprintf(buf, "%s{cache=\"%s\"} %g\n\n", metricName, p.cacheName, value)
}

// addGaugeWithLabels 添加带标签的仪表类型指标
func (p *PrometheusExporter) addGaugeWithLabels(buf *bytes.Buffer, name, help string, value float64, labels string) {
	metricName := fmt.Sprintf("%s_%s", p.prefix, name)
	fmt.Fprintf(buf, "# HELP %s %s\n", metricName, help)
	fmt.Fprintf(buf, "# TYPE %s gauge\n", metricName)
	fmt.Fprintf(buf, "%s{%s} %g\n\n", metricName, labels, value)
}

// addHistogram 添加直方图类型指标
func (p *PrometheusExporter) addHistogram(buf *bytes.Buffer, name, help string, histogram *HistogramSnapshot) {
	metricName := fmt.Sprintf("%s_%s", p.prefix, name)
	fmt.Fprintf(buf, "# HELP %s %s\n", metricName, help)
	fmt.Fprintf(buf, "# TYPE %s histogram\n", metricName)

	// 添加桶
	cumulativeCount := uint64(0)
	for i, count := range histogram.BucketCounts {
		cumulativeCount += count
		bucketBound := histogram.BucketBounds[i]
		fmt.Fprintf(buf, "%s_bucket{cache=\"%s\",le=\"%d\"} %d\n",
			metricName, p.cacheName, bucketBound, cumulativeCount)
	}

	// 添加+Inf桶
	fmt.Fprintf(buf, "%s_bucket{cache=\"%s\",le=\"+Inf\"} %d\n",
		metricName, p.cacheName, histogram.Count)

	// 添加总和和计数
	fmt.Fprintf(buf, "%s_sum{cache=\"%s\"} %d\n", metricName, p.cacheName, histogram.Sum)
	fmt.Fprintf(buf, "%s_count{cache=\"%s\"} %d\n\n", metricName, p.cacheName, histogram.Count)
}

// ServeHTTP 实现http.Handler接口，用于提供Prometheus指标端点
func (p *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(p.Export()))
}

// RegisterWithPrometheus 注册到Prometheus客户端库
// 注意：此方法需要在导入Prometheus客户端库时使用
// 这里只提供接口定义，实际实现需要在使用时根据需要添加
func (p *PrometheusExporter) RegisterWithPrometheus() error {
	// 此方法需要在项目中导入Prometheus客户端库时实现
	// 例如：
	// import "github.com/prometheus/client_golang/prometheus"
	//
	// 实现可能如下：
	// registry := prometheus.NewRegistry()
	// collector := NewPrometheusCollector(p)
	// registry.MustRegister(collector)
	// ...
	return nil
}

// ExposePrometheusMetrics 导出Prometheus格式的指标
// 这是一个便捷方法，用于直接获取Prometheus格式的指标字符串
func ExposePrometheusMetrics(metrics *Metrics, cacheName string) string {
	exporter := NewPrometheusExporter(metrics, cacheName)
	return exporter.Export()
}
