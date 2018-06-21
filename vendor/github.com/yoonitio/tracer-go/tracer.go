package tracer

import (
	"github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
	"github.com/yoonitio/tracer-go/recorders"
	"os"
)

type (
	TracerOption interface {
		Apply(*Tracer)
	}

	Tracer struct {
		opentracing.Tracer
		path string
		name string
		streamName string
	}

	PathOption struct {
		path string
	}

	NameOption struct {
		name string
	}

	StreamNameOption struct {
		streamName string
	}
)

const DefaultTracePathEnvKey = "YOONIT_TRACE_PATH"
const DefaultServiceNameEnvKey = "YOONIT_SERVICE_NAME"
const DefaultStreamNameEnvKey = "YOONIT_STREAM_NAME"

func NewTracer(opts ...TracerOption) opentracing.Tracer {
	tracer := &Tracer{
		path: os.Getenv(DefaultTracePathEnvKey),
		name: os.Getenv(DefaultServiceNameEnvKey),
		streamName: os.Getenv(DefaultStreamNameEnvKey),
	}

	for _, o := range opts {
		o.Apply(tracer)
	}

	recorder := recorders.NewDummyRecorder()
	if tracer.streamName != "" {
		recorder = recorders.NewKinesisRecorder(tracer.streamName)
	} else if tracer.path != "" {
		recorder = recorders.NewFileRecorder(tracer.path)
	}

	tracer.Tracer = basictracer.NewWithOptions(basictracer.Options{
		ShouldSample:   func(traceID uint64) bool { return true },
		MaxLogsPerSpan: 100,
		Recorder:       recorder,
	})

	return tracer
}

func (t *Tracer) StartSpan(
	operationName string,
	opts ...opentracing.StartSpanOption,
) opentracing.Span {
	if t.name != "" {
		opts = append(opts, opentracing.Tag{Key: "service", Value: t.name})
	}
	return t.Tracer.StartSpan(operationName, opts...)
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

func (o NameOption) Apply(t *Tracer) {
	t.name = o.name
}

func WithName(name string) TracerOption {
	return NameOption{name: name}
}

func (o StreamNameOption) Apply(t *Tracer) {
	t.streamName = o.streamName
}

func WithStreamName(streamName string) TracerOption {
	return StreamNameOption{streamName: streamName}
}
