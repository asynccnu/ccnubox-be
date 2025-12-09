package htmlx

import (
	"bytes"
	"io"
)

type FileToHTMLConverter interface {
	ConvertToHTML(reader io.Reader) (string, error)
}

// DocxToHTMLConverter 实现了FileToHTMLConverter接口，用于将Docx文件转换为HTML
type DocxToHTMLConverter struct{}

func (d *DocxToHTMLConverter) ConvertToHTML(reader io.Reader) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return "", err
	}
	// TODO implement me
	panic("implement me")
}
