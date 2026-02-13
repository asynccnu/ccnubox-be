package data

import (
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/test"
	"strconv"
	"testing"
	"time"
)


func TestDelayKafka(t *testing.T) {
	brokers := []string{"localhost:9092"}
	kpb := NewKafkaProducerBuilder(&conf.Data{
		Kafka: &conf.Data_Kafka{
			Brokers: brokers,
		},
	})

	kcb := NewKafkaConsumerBuilder(&conf.Data{
		Kafka: &conf.Data_Kafka{
			Brokers: brokers,
		},
	})

	dk, _, err := NewDelayKafka(kpb, kcb, DelayKafkaConfig{
		delayTopic: "test-delay",
		realTopic:  "test-real",
		delayTime:  5 * time.Second,
	}, test.NewLogger())
	if err != nil {
		t.Fatalf("failed to create DelayKafka: %v", err)
	}

	go func() {
		if err1 := dk.Consume("be-classlist-real", func(key, value []byte) {
			t.Logf("key:%v,value:%v", string(key), string(value))
		}); err1 != nil {
			t.Errorf("failed to consume: %v", err1)
		}
	}()

	// 等待消费者初始化
	time.Sleep(2 * time.Second)

	for i := 0; i < 4; i++ {
		key := "test" + fmt.Sprintf("%d", time.Now().UnixMilli())
		val := "test" + strconv.Itoa(i)
		if err1 := dk.Send([]byte(key), []byte(val)); err1 != nil {
			t.Errorf("failed to send: %v", err1)
		}
		t.Logf("send message key:%v,val:%v,time:%v", key, val, time.Now().Format("2006-01-02 15:04:05"))
	}

	time.Sleep(10 * time.Second)
	dk.Close()
}
