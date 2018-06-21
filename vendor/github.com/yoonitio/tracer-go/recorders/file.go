package recorders

import (
	"encoding/json"
	"fmt"
	"github.com/opentracing/basictracer-go"
	"log"
	"os"
	"path"
)

type FileRecorder struct {
	path string
}

func NewFileRecorder(path string) basictracer.SpanRecorder {
	os.MkdirAll(path, 0755)
	return &FileRecorder{path: path}
}

func (r *FileRecorder) RecordSpan(span basictracer.RawSpan) {
	dir := path.Join(r.path, fmt.Sprintf("%d", span.Context.TraceID))
	os.MkdirAll(dir, 0755)
	file, _ := os.Create(path.Join(dir, fmt.Sprintf("%d.span.json", span.Context.SpanID)))
	defer file.Close()

	var logs []Logs
	for _, l := range span.Logs {
		logs = append(logs, Logs{Timestamp: l.Timestamp, Fields: Materialize(l.Fields)})
	}

	b, err := json.Marshal(&Span{
		RawSpan: span,
		Logs:    logs,
	})
	if err != nil {
		log.Printf("error: %v\n", err)
	}
	file.Write(b)
}
