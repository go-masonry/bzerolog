package bzerolog

import (
	"context"
	"fmt"
	"time"

	"github.com/go-masonry/mortar/interfaces/log"
	"github.com/rs/zerolog"
)

type zerologEntryWrapper struct {
	err                 error
	fields              map[string]interface{}
	rootWrapper         *zerologWrapper
	calledWithSomeField bool
}

func newEntry(root *zerologWrapper, calledWithSomeField bool, staticFields map[string]interface{}) *zerologEntryWrapper {
	// copy map
	fields := make(map[string]interface{})
	for k, v := range staticFields {
		fields[k] = v
	}
	return &zerologEntryWrapper{
		err:                 nil,
		fields:              fields,
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
		zew.Error(ctx, format, args...)
	case log.WarnLevel:
		zew.Warn(ctx, format, args...)
	case log.InfoLevel:
		zew.Info(ctx, format, args...)
	case log.DebugLevel:
		zew.Debug(ctx, format, args...)
	default:
		zew.Trace(ctx, format, args...)
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

func (zew *zerologEntryWrapper) msg(ctx context.Context, level zerolog.Level, format string, args ...interface{}) {
	zew.extractFromContextIfNeeded(ctx)
	event := zew.rootWrapper.instance.WithLevel(level)
	event = zew.addTimestampIfNeeded(event)
	event = zew.includeCallerIfNeeded(event)
	event = event.AnErr(zerolog.ErrorFieldName, zew.err)
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

func (zew *zerologEntryWrapper) extractFromContextIfNeeded(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			zew.WithField("__panic__", fmt.Sprintf("one of the context extractors panicked: %v", r))
		}
	}()
	if ctx != nil && len(zew.rootWrapper.cfg.contextExtractors) > 0 {
		for _, extractor := range zew.rootWrapper.cfg.contextExtractors {
			fields := extractor(ctx)
			// merge maps
			for name, value := range fields {
				zew.WithField(name, value)
			}
		}
	}
}

// Sanity
var _ log.Fields = (*zerologEntryWrapper)(nil)
