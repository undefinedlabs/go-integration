package recorders

import (
	"encoding/json"
	"fmt"
	"github.com/opentracing/basictracer-go"
	"log"
	"os"
	"path"
	"time"
)

type FileRecorder struct {
	path string
}

type JSONLogs struct {
	Timestamp time.Time
	Fields    map[string]interface{}
}

type JSONSpan struct {
	basictracer.RawSpan
	Logs []JSONLogs
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

	var logs []JSONLogs
	for _, l := range span.Logs {
		logs = append(logs, JSONLogs{Timestamp: l.Timestamp, Fields: Materialize(l.Fields)})
	}

	b, err := json.Marshal(&JSONSpan{
		RawSpan: span,
		Logs:    logs,
	})
	if err != nil {
		log.Printf("error: %v\n", err)
	}
	file.Write(b)
}
