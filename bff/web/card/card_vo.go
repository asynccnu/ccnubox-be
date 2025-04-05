package card

import (
	"time"
)

type NoteUserKeyRequest struct {
	Key string `json:"key"`
}

type UpdateUserKeyRequest struct {
	Key string `json:"key"`
}

type GetRecordOfConsumptionRequest struct {
	Key       string `json:"key"`
	StartTime string `json:"start_time"`
	Type      string `json:"type"`
}

type GetRecordOfConsumptionResponse struct {
	Records []Records
}

type Records struct {
	SMT_TIMES        uint32    `protobuf:"varint,1,opt,name=SMT_TIMES,json=SMTTIMES,proto3" json:"SMT_TIMES,omitempty"`
	SMT_DEALDATETIME time.Time `protobuf:"bytes,2,opt,name=SMT_DEALDATETIME,json=SMTDEALDATETIME,proto3" json:"SMT_DEALDATETIME,omitempty"`
	SMT_ORG_NAME     string    `protobuf:"bytes,3,opt,name=SMT_ORG_NAME,json=SMTORGNAME,proto3" json:"SMT_ORG_NAME,omitempty"`
	SMT_DEALNAME     string    `protobuf:"bytes,4,opt,name=SMT_DEALNAME,json=SMTDEALNAME,proto3" json:"SMT_DEALNAME,omitempty"`
	AfterMoney       float32   `protobuf:"fixed32,5,opt,name=after_money,json=afterMoney,proto3" json:"after_money,omitempty"`
	Money            float32   `protobuf:"fixed32,6,opt,name=money,proto3" json:"money,omitempty"`
}

// Result 你可以通过在 Result 里面定义更加多的字段，来配合 Wrap 方法
type Result struct {
	Code int    `json:"code"` // 错误码，非 0 表示失败
	Msg  string `json:"msg"`  // 错误或成功 描述
	Data any    `json:"data"`
}
type GetRecordsResp struct {
	Records []Records `json:"records"`
}
