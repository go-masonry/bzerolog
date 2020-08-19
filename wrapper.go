package bzerolog

import (
	"context"

	"github.com/go-masonry/mortar/interfaces/log"
	"github.com/rs/zerolog"
)

type zerologWrapper struct {
	cfg      *zerologConfig
	instance zerolog.Logger
}

func newWrapper(cfg *zerologConfig, instance zerolog.Logger) *zerologWrapper {
	return &zerologWrapper{
		cfg:      cfg,
		instance: instance,
	}
}

// Highly detailed tracing messages. Produces the most voluminous output. Used by developers for developers
// Some implementations doesn't have that granularity and use Debug level instead
func (zw *zerologWrapper) Trace(ctx context.Context, format string, args ...interface{}) {
	newEntry(zw, false, zw.cfg.staticFields).Trace(ctx, format, args...)
}

// Relatively detailed tracing messages. Used mostly by developers to debug the flow
func (zw *zerologWrapper) Debug(ctx context.Context, format string, args ...interface{}) {
	newEntry(zw, false, zw.cfg.staticFields).Debug(ctx, format, args...)
}

// Informational messages that might make sense to users unfamiliar with this application
func (zw *zerologWrapper) Info(ctx context.Context, format string, args ...interface{}) {
	newEntry(zw, false, zw.cfg.staticFields).Info(ctx, format, args...)
}

// Potentially harmful situations of interest to users that indicate potential problems.
func (zw *zerologWrapper) Warn(ctx context.Context, format string, args ...interface{}) {
	newEntry(zw, false, zw.cfg.staticFields).Warn(ctx, format, args...)
}

// Very severe error events that might cause the application to terminate or misbehave.
// It's not intended to use to log every 'error', for that use 'WithError(err).<Trace|Debug|Info|...>(...)'
func (zw *zerologWrapper) Error(ctx context.Context, format string, args ...interface{}) {
	newEntry(zw, false, zw.cfg.staticFields).Error(ctx, format, args...)
}

func (zw *zerologWrapper) Custom(ctx context.Context, level log.Level, skipAdditionalFrames int, format string, args ...interface{}) {
	newEntry(zw, false, zw.cfg.staticFields).Custom(ctx, level, skipAdditionalFrames, format, args...)
}

// Add an error to the log structure, output depends on the implementation
func (zw *zerologWrapper) WithError(err error) log.Fields {
	return newEntry(zw, true, zw.cfg.staticFields).WithError(err)
}

// Add an informative field to the log structure, output depends on the implementation
func (zw *zerologWrapper) WithField(name string, value interface{}) log.Fields {
	return newEntry(zw, true, zw.cfg.staticFields).WithField(name, value)
}

// Implementor returns the actual lib/struct that is responsible for the above logic
func (zw *zerologWrapper) Configuration() log.LoggerConfiguration {
	return zw
}

// Implement Configuration
func (zw *zerologWrapper) Level() log.Level {
	return zw.cfg.level
}

func (zw *zerologWrapper) Implementation() interface{} {
	return zw.instance
}

// Sanity
var _ log.Logger = (*zerologWrapper)(nil)
