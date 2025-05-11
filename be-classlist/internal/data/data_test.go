package data

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKafkaProducerBuilder_Build(t *testing.T) {
	type fields struct {
		brokers []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "len(brokers) is 0",
			fields:  fields{brokers: []string{}},
			wantErr: assert.Error,
		},
		{
			name:    "brokers is not right",
			fields:  fields{brokers: []string{"localhost:9194"}},
			wantErr: assert.Error,
		},
		{
			name:    "brokers is right",
			fields:  fields{brokers: []string{"localhost:9094"}},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := KafkaProducerBuilder{
				brokers: tt.fields.brokers,
			}
			_, err := pb.Build()
			if !tt.wantErr(t, err, fmt.Sprintf("Build()")) {
				return
			}
		})
	}
}

func TestKafkaConsumerBuilder_Build(t *testing.T) {
	type fields struct {
		brokers []string
	}
	type args struct {
		groupID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "len(brokers) is 0",
			fields:  fields{brokers: []string{}},
			args:    args{groupID: "test"},
			wantErr: assert.Error,
		},
		{
			name:    "brokers is not right",
			fields:  fields{brokers: []string{"localhost:9194"}},
			args:    args{groupID: "test"},
			wantErr: assert.Error,
		},
		{
			name:    "brokers is right",
			fields:  fields{brokers: []string{"localhost:9094"}},
			args:    args{groupID: "test"},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := KafkaConsumerBuilder{
				brokers: tt.fields.brokers,
			}
			_, err := cb.Build(tt.args.groupID)
			if !tt.wantErr(t, err, fmt.Sprintf("Build(%v)", tt.args.groupID)) {
				return
			}
		})
	}
}
