package bzerolog

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"

	"container/list"

	"github.com/go-masonry/mortar/interfaces/log"
)

// Defaults

// ConsoleWriter writes to console in Human Readable format
//
// Note:
// 		It's important NOT to exclude time when using this writer
func ConsoleWriter(writer ...io.Writer) io.Writer {
	if len(writer) > 0 {
		return zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = writer[0]
		})
	}
	return zerolog.NewConsoleWriter()
}

const (
	skipWrapperFrames = 1
)

// ZerologBuilder is helper builder to configure zerolog instance
type ZerologBuilder interface {
	log.Builder
	// SetWriter set where output should be printed
	SetWriter(writer io.Writer) ZerologBuilder
	// AddStaticFields will always add provided fields to every log entry
	AddStaticFields(fields map[string]interface{}) ZerologBuilder
	// ExcludeTime configures zerolog to exclude any time field
	ExcludeTime() ZerologBuilder
	// SetCustomTimeFormatter sets the time format field, have no effect if ExcludeTime is called
	SetCustomTimeFormatter(format string) ZerologBuilder
	// IncludeCaller adds caller:line to log entry
	IncludeCaller() ZerologBuilder
}

type zerologConfig struct {
	writer            io.Writer
	level             log.Level
	staticFields      map[string]interface{}
	contextExtractors []log.ContextExtractor
	excludeTimeField  bool
	customTimeFormat  string
	includeCaller     bool
	skipCallerFrames  int
}

type zerologBuilder struct {
	ll *list.List
}

// Builder creates a new zerolog builder
func Builder() ZerologBuilder {
	return &zerologBuilder{
		ll: list.New(),
	}
}

func (zb *zerologBuilder) SetWriter(writer io.Writer) ZerologBuilder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		cfg.writer = writer
	})
	return zb
}

func (zb *zerologBuilder) SetLevel(level log.Level) log.Builder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		cfg.level = level
	})
	return zb
}

func (zb *zerologBuilder) AddStaticFields(fields map[string]interface{}) ZerologBuilder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		for k, v := range fields {
			cfg.staticFields[k] = v
		}
	})
	return zb
}

func (zb *zerologBuilder) ExcludeTime() ZerologBuilder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		cfg.excludeTimeField = true
	})
	return zb
}

func (zb *zerologBuilder) SetCustomTimeFormatter(format string) ZerologBuilder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		cfg.customTimeFormat = format
	})
	return zb
}

func (zb *zerologBuilder) IncludeCaller() ZerologBuilder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		cfg.includeCaller = true
	})
	return zb
}

func (zb *zerologBuilder) IncrementSkipFrames(skip int) log.Builder {
	zb.ll.PushBack(func(cfg *zerologConfig) {
		cfg.skipCallerFrames += skip
	})
	return zb
}

func (zb *zerologBuilder) Build() log.Logger {
	config := &zerologConfig{
		writer:            os.Stderr,
		level:             log.TraceLevel,
		staticFields:      make(map[string]interface{}),
		contextExtractors: nil,
		customTimeFormat:  time.RFC3339,
		excludeTimeField:  false,
		skipCallerFrames:  zerolog.CallerSkipFrameCount + skipWrapperFrames, // bzerolog will add it's own amount of frames to skip and so do we
		includeCaller:     false,
	}
	// Purely sanity code that should not be ever...
	if zb == nil {
		return newWrapper(config, zerolog.New(config.writer))
	}
	for e := zb.ll.Front(); e != nil; e = e.Next() {
		f := e.Value.(func(cfg *zerologConfig))
		f(config)
	}
	zLevel, _ := zerolog.ParseLevel(config.level.String())
	logContext := zerolog.New(config.writer).Level(zLevel).With()
	return newWrapper(config, logContext.Logger())
}

// Sanity
var _ log.Builder = (*zerologBuilder)(nil)
