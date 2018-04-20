package tracer

import (
	"github.com/opentracing/opentracing-go/log"
)

type fieldsAsMap map[string]interface{}

func Materialize(logFields []log.Field) map[string]interface{} {
	fields := fieldsAsMap(make(map[string]interface{}, len(logFields)))
	for _, field := range logFields {
		field.Marshal(fields)
	}
	return fields
}

func (ml fieldsAsMap) EmitString(key, value string) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitBool(key string, value bool) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitInt(key string, value int) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitInt32(key string, value int32) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitInt64(key string, value int64) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitUint32(key string, value uint32) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitUint64(key string, value uint64) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitFloat32(key string, value float32) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitFloat64(key string, value float64) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitObject(key string, value interface{}) {
	ml[key] = value
}

func (ml fieldsAsMap) EmitLazyLogger(value log.LazyLogger) {
	value(ml)
}
