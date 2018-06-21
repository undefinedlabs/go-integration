package recorders

import (
	"log"
	"github.com/opentracing/basictracer-go"
	"github.com/vmihailenco/msgpack"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/aws/session"
	"strconv"
)

type KinesisRecorder struct {
	streamName string
}

func NewKinesisRecorder(streamName string) basictracer.SpanRecorder {
	return &KinesisRecorder{streamName: streamName}
}

func (r *KinesisRecorder) RecordSpan(span basictracer.RawSpan) {
	var logs []Logs
	for _, l := range span.Logs {
		logs = append(logs, Logs{Timestamp: l.Timestamp, Fields: Materialize(l.Fields)})
	}

	b, err := msgpack.Marshal(&Span{
		RawSpan: span,
		Logs:    logs,
	})
	if err != nil {
		log.Printf("error: %v\n", err)
	}

	partitionKey := strconv.Itoa(int(span.Context.TraceID))
	streamName := r.streamName
	svc := kinesis.New(session.Must(session.NewSession()))
	svc.PutRecord(&kinesis.PutRecordInput{
		Data: b,
		PartitionKey: &partitionKey,
		StreamName: &streamName,
	})
}
