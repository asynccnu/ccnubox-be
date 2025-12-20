package wrr

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"sync"
)

const WeightRoundRobin = "custom_weighted_round_robin"

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(WeightRoundRobin,
		&WeightedPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

type WeightedPicker struct {
	mutex sync.Mutex
	conns []*weightConn
}

func (b *WeightedPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(b.conns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	// 这里实时计算 totalWeight 是为了方便你作业动态调整权重
	var totalWeight int
	var res *weightConn

	b.mutex.Lock()
	for _, node := range b.conns {
		totalWeight += node.weight
		node.currentWeight += node.weight
		if res == nil || res.currentWeight < node.currentWeight {
			res = node
		}
	}
	res.currentWeight -= totalWeight
	b.mutex.Unlock()
	return balancer.PickResult{
		SubConn: res.SubConn,
		Done: func(info balancer.DoneInfo) {
			// 在这里执行 failover 有关的事情
			// 例如说把 res 的 currentWeight 进一步调低到一个非常低的值
			// 也可以直接把 res 从 b.conns 删除
		},
	}, nil
}

type WeightedPickerBuilder struct {
}

func (b *WeightedPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	conns := make([]*weightConn, 0, len(info.ReadySCs))
	for con, conInfo := range info.ReadySCs {
		// 如果不存在，那么权重就是 0
		weightVal, _ := conInfo.Address.Metadata.(map[string]any)["weight"]
		// 经过注册中心的转发之后，变成了 float64，要小心这个问题
		weight, _ := weightVal.(float64)
		conns = append(conns, &weightConn{
			SubConn:       con,
			weight:        int(weight),
			currentWeight: int(weight),
		})
	}
	return &WeightedPicker{
		conns: conns,
	}
}

type weightConn struct {
	// 初始权重
	weight int
	// 当前权重
	currentWeight int
	balancer.SubConn
}
