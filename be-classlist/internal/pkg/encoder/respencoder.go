package encoder

import (
	"encoding/json"
	"github.com/go-kratos/kratos/v2/encoding"
	"google.golang.org/protobuf/proto"
	nethttp "net/http"
)

// 返回的结构
type Response struct {
	Code    int         `json:"code"`
	Reason  string      `json:"reason"`
	Message interface{} `json:"message"`
}

func RespEncoder(w nethttp.ResponseWriter, r *nethttp.Request, i interface{}) error {
	codec := encoding.GetCodec("json")
	messageMap := make(map[string]interface{})
	messageStr, _ := codec.Marshal(i.(proto.Message))
	_ = codec.Unmarshal(messageStr, &messageMap)

	// Notice 加上下面的逻辑，如果返回的结果中只有一个字段，那会将结果直接返回到message中
	// Notice 看实际情况，一般不会这样做的
	/*
			比如：返回的结果是："desc":{"name":"whw"}，只有desc一个字段
			那么 结果会是这样:
			{
		      "code": 200,
		      "reason": "",
		      "message": {"name": "whw"}
			}
	*/
	//if len(messageMap) == 1 {
	//	for _, v := range messageMap {
	//		i = v
	//	}
	//}

	resp := Response{
		Code:   200,
		Reason: "",
	}

	if msg, ok := messageMap["message"]; ok {
		i = msg
	}

	message, err := codec.Marshal(i)
	_ = json.Unmarshal(message, &resp.Message)
	if err != nil {
		return err
	}

	data, err := codec.Marshal(resp)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
	return nil
}
