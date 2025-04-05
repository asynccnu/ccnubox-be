package qiniu

import (
	"bytes"
	"context"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

type QiniuClient interface {
	Upload(fileBytes []byte, fileName string) (string, error)
}

type qiniuClient struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Bucket    string `yaml:"bucket"`
	Domain    string `yaml:"domain"`
	BaseName  string `yaml:"baseName"`
}

func NewQiniuService(
	AccessKey string,
	SecretKey string,
	Bucket string,
	Domain string,
	BaseName string,
) QiniuClient {
	return &qiniuClient{
		AccessKey: AccessKey,
		SecretKey: SecretKey,
		Bucket:    Bucket,
		Domain:    Domain,
		BaseName:  BaseName,
	}
}

func (s *qiniuClient) Upload(fileBytes []byte, filename string) (string, error) {
	mac := qbox.NewMac(s.AccessKey, s.SecretKey)
	fileName := s.BaseName + filename
	putPolicy := storage.PutPolicy{
		Scope: s.Bucket + ":" + fileName, // 设置为 "bucket:fileName" 形式以支持覆盖同名文件
	}

	uploadToken := putPolicy.UploadToken(mac)

	cfg := storage.Config{
		Zone:          &storage.ZoneHuadong, // 存储在华东区
		UseHTTPS:      true,
		UseCdnDomains: false,
	}

	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}

	err := formUploader.Put(context.Background(), &ret, uploadToken, fileName, bytes.NewReader(fileBytes), int64(len(fileBytes)), nil)
	if err != nil {
		return "", err
	}

	// 返回文件的完整 URL
	return s.Domain + "/" + ret.Key, nil
}
