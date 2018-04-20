package tracer

import (
	"github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
	"os"
)

type (
	TracerOption interface {
		Apply(*Tracer)
	}

	Tracer struct {
		opentracing.Tracer
		path string
	}

	PathOption struct {
		path string
	}
)

const DefaultTracePathEnvKey = "YOONIT_TRACE_PATH"

func NewTracer(opts ...TracerOption) opentracing.Tracer {
	tracer := &Tracer{
		path: os.Getenv(DefaultTracePathEnvKey),
	}

	for _, o := range opts {
		o.Apply(tracer)
	}

	recorder := NewDummyRecorder()
	if tracer.path != "" {
		recorder = NewFileRecorder(tracer.path)
	}

	tracer.Tracer = basictracer.NewWithOptions(basictracer.Options{
		ShouldSample:   func(traceID uint64) bool { return true },
		MaxLogsPerSpan: 100,
		Recorder:       recorder,
	})

	return tracer
}

func (t *Tracer) Path() string {
	return t.path
}

func (o PathOption) Apply(t *Tracer) {
	t.path = o.path
}

func WithPath(path string) TracerOption {
	return PathOption{path: path}
}
