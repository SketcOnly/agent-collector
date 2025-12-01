package metrics

// MetricFactory 指标工厂，用于统一创建指标（counter/gauge/histogram）。
type MetricFactory struct {
	reg Registers
}

// NewMetricFactory 创建指标工厂
func NewMetricFactory(reg Registers) *MetricFactory {
	return &MetricFactory{reg: reg}
}
