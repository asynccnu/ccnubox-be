package prometheusx

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus 是一个 Prometheus 工具包
type Prometheus struct {
	namespace  string
	subsystem  string
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	lock       sync.RWMutex
}

type PrometheusCounter struct {
	RouterCounter     *prometheus.CounterVec
	ActiveConnections *prometheus.GaugeVec
	DurationTime      *prometheus.HistogramVec
}

// NewPrometheus 创建一个新的 Prometheus 工具包实例
func NewPrometheus(namespace string) *Prometheus {
	return &Prometheus{
		namespace:  namespace,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

// RegisterCounter 注册一个 Counter 指标
func (p *Prometheus) RegisterCounter(name, help string, labels []string) *prometheus.CounterVec {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, exists := p.counters[name]; exists {
		return p.counters[name]
	}

	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: p.namespace,
		Subsystem: p.subsystem,
		Name:      name,
		Help:      help,
	}, labels)
	prometheus.MustRegister(counter)
	p.counters[name] = counter
	return counter
}

// RegisterGauge 注册一个 Gauge 指标
func (p *Prometheus) RegisterGauge(name, help string, labels []string) *prometheus.GaugeVec {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, exists := p.gauges[name]; exists {
		return p.gauges[name]
	}

	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: p.namespace,
		Subsystem: p.subsystem,
		Name:      name,
		Help:      help,
	}, labels)
	prometheus.MustRegister(gauge)
	p.gauges[name] = gauge
	return gauge
}

// RegisterHistogram 注册一个 Histogram 指标
func (p *Prometheus) RegisterHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, exists := p.histograms[name]; exists {
		return p.histograms[name]
	}

	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: p.namespace,
		Subsystem: p.subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labels)
	prometheus.MustRegister(histogram)
	p.histograms[name] = histogram
	return histogram
}

// GetCounter 获取已注册的 Counter
func (p *Prometheus) GetCounter(name string) *prometheus.CounterVec {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.counters[name]
}

// GetGauge 获取已注册的 Gauge
func (p *Prometheus) GetGauge(name string) *prometheus.GaugeVec {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.gauges[name]
}

// GetHistogram 获取已注册的 Histogram
func (p *Prometheus) GetHistogram(name string) *prometheus.HistogramVec {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.histograms[name]
}
