package bzerolog

import (
	"context"
	"time"

	"github.com/go-masonry/mortar/interfaces/log"
	"github.com/rs/zerolog"
)

type zerologEntryWrapper struct {
	err                 error
	staticFields        map[string]interface{}
	fields              map[string]interface{}
	rootWrapper         *zerologWrapper
	calledWithSomeField bool
}

func newEntry(root *zerologWrapper, calledWithSomeField bool, staticFields map[string]interface{}) *zerologEntryWrapper {
	// copy map
	fields := make(map[string]interface{})
	return &zerologEntryWrapper{
		err:                 nil,
		fields:              fields,
		staticFields:        staticFields,
		rootWrapper:         root,
		calledWithSomeField: calledWithSomeField,
	}
}

func (zew *zerologEntryWrapper) Trace(ctx context.Context, format string, args ...interface{}) {
	zew.msg(ctx, zerolog.TraceLevel, format, args...)
}
func (zew *zerologEntryWrapper) Debug(ctx context.Context, format string, args ...interface{}) {
	zew.msg(ctx, zerolog.DebugLevel, format, args...)
}
func (zew *zerologEntryWrapper) Info(ctx context.Context, format string, args ...interface{}) {
	zew.msg(ctx, zerolog.InfoLevel, format, args...)
}
func (zew *zerologEntryWrapper) Warn(ctx context.Context, format string, args ...interface{}) {
	zew.msg(ctx, zerolog.WarnLevel, format, args...)
}
func (zew *zerologEntryWrapper) Error(ctx context.Context, format string, args ...interface{}) {
	zew.msg(ctx, zerolog.ErrorLevel, format, args...)
}
func (zew *zerologEntryWrapper) Custom(ctx context.Context, level log.Level, format string, args ...interface{}) {
	switch level {
	case log.ErrorLevel:
		zew.msg(ctx, zerolog.ErrorLevel, format, args...)
	case log.WarnLevel:
		zew.msg(ctx, zerolog.WarnLevel, format, args...)
	case log.InfoLevel:
		zew.msg(ctx, zerolog.InfoLevel, format, args...)
	case log.DebugLevel:
		zew.msg(ctx, zerolog.DebugLevel, format, args...)
	default:
		zew.msg(ctx, zerolog.TraceLevel, format, args...)
	}
}

// Add an error to the log structure, output depends on the implementation
func (zew *zerologEntryWrapper) WithError(err error) log.Fields {
	zew.err = err
	return zew
}

// Add an informative field to the log structure, output depends on the implementation
func (zew *zerologEntryWrapper) WithField(name string, value interface{}) log.Fields {
	zew.fields[name] = value
	return zew
}

func (zew *zerologEntryWrapper) msg(_ context.Context, level zerolog.Level, format string, args ...interface{}) {
	event := zew.rootWrapper.instance.WithLevel(level)
	event = zew.addTimestampIfNeeded(event)
	event = zew.includeCallerIfNeeded(event)
	event = event.AnErr(zerolog.ErrorFieldName, zew.err)
	if len(zew.staticFields) > 0 {
		event = event.Fields(zew.staticFields)
	}
	if len(zew.fields) > 0 {
		event = event.Fields(zew.fields)
	}
	// Write
	if len(args) > 0 {
		event.Msgf(format, args...)
	} else {
		event.Msg(format)
	}
}

func (zew *zerologEntryWrapper) includeCallerIfNeeded(e *zerolog.Event) *zerolog.Event {
	if zew.rootWrapper.cfg.includeCaller {
		skip := zew.rootWrapper.cfg.skipCallerFrames
		if !zew.calledWithSomeField {
			skip++ // This log was first called with Error/Field and then Debug/Trace/...
		}
		return e.Caller(skip)
	}
	return e
}

func (zew *zerologEntryWrapper) addTimestampIfNeeded(e *zerolog.Event) *zerolog.Event {
	if !zew.rootWrapper.cfg.excludeTimeField {
		now := time.Now()
		// The reason we use this approach is that we don't want to override global bzerolog variables
		return e.Str(zerolog.TimestampFieldName, now.Format(zew.rootWrapper.cfg.customTimeFormat))
	}
	return e
}

// Sanity
var _ log.Fields = (*zerologEntryWrapper)(nil)
