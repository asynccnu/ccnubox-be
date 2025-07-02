package data

type DelayKafka struct{}

// 不直接生产 biz.DelayQueue
// 是把生产函数彻底接头化
// 在 wire.go 中建议更换队列的实现
func NewDelayKafka() *DelayKafka {
	return &DelayKafka{}
}

func (d *DelayKafka) Send(key, value []byte) error {
	return nil
}

func (d *DelayKafka) Consume(groupID string, f func(key, value []byte)) error {
	return nil
}

func (d *DelayKafka) Close() {}
