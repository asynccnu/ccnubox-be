package service

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// 模拟的 FreeClassRoomSaver 实现，用于测试
type MockFreeClassRoomBiz struct{}

func (m *MockFreeClassRoomBiz) SaveFreeClassRoomInfo(ctx context.Context, year, semester string, cwtPairs []model.CTWPair) error {
	file, err := os.Create("data.json")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	//encoder.SetIndent("", "  ") // 设置缩进使得JSON更具可读性
	err = encoder.Encode(&cwtPairs)
	if err != nil {
		return err
	}
	return nil
}

func TestUploadSelection(t *testing.T) {
	// 模拟上传的 JSON 数据
	jsonData := `{
		"year": "2024",
		"semester": "2",
		"sheets": {
			"2024级": {
				"class_time_idx": 7,
				"class_where_idx": 8
			},
			"2023级" : {
				"class_time_idx": 7,
				"class_where_idx": 8
			},
			"2022级" : {
				"class_time_idx": 7,
				"class_where_idx": 8
			},
			"2021级" : {
				"class_time_idx": 7,
				"class_where_idx": 8
			},
			"公共课" : {
				"class_time_idx": 7,
				"class_where_idx": 8
			}
		}
	}`

	excelFile, err := os.OpenFile("./2024-2025学年第1学期选课手册.xlsx", os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to create test Excel file: %v", err)
	}

	// 创建请求体（multipart/form-data）
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// 添加 JSON 数据部分
	part, err := writer.CreateFormField("json_data")
	if err != nil {
		t.Fatalf("Failed to create form field for json_data: %v", err)
	}
	part.Write([]byte(jsonData))

	// 添加文件部分
	filePart, err := writer.CreateFormFile("file", "test.xlsx")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	_, err = io.Copy(filePart, excelFile)
	if err != nil {
		t.Fatalf("Failed to copy Excel file data: %v", err)
	}

	// 结束 multipart 编写
	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// 创建模拟请求
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 使用 httptest.NewRecorder() 来模拟响应
	rr := httptest.NewRecorder()

	// 创建服务和处理器
	selectionUploader := &SelectionUploader{
		freeClassRoom: &MockFreeClassRoomBiz{},
	}

	// 调用 UploadSelection 方法
	selectionUploader.UploadSelection(rr, req)

	t.Log(rr.Body.String())
	// 检查响应状态
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200")
}
