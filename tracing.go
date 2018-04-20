package integration

import (
	"github.com/opentracing/opentracing-go"
	"github.com/yoonitio/tracer-go"
)

var tracePath string

func init() {
	t := tracer.NewTracer()
	opentracing.SetGlobalTracer(t)
	tracePath = t.(*tracer.Tracer).Path()
}
