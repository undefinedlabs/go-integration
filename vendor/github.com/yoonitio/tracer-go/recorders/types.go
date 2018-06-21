package recorders

import (
	"github.com/opentracing/basictracer-go"
	"time"
)

type Logs struct {
	Timestamp time.Time
	Fields    map[string]interface{}
}

type Span struct {
	basictracer.RawSpan
	Logs []Logs
}
