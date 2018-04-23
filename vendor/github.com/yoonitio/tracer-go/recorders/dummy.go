package recorders

import (
	"github.com/opentracing/basictracer-go"
)

type DummyRecorder struct{}

func NewDummyRecorder() basictracer.SpanRecorder {
	return &DummyRecorder{}
}

func (r *DummyRecorder) RecordSpan(span basictracer.RawSpan) {}
