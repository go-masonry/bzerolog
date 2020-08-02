package bzerolog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-masonry/mortar/interfaces/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type wrapperTestSuite struct {
	suite.Suite
	buf    *bytes.Buffer
	logger log.Logger
}

func TestWrapper(t *testing.T) {
	suite.Run(t, new(wrapperTestSuite))
}

func TestDefaultConfiguration(t *testing.T) {
	logger := Builder().Build()
	configuration := logger.Configuration()
	assert.Equal(t, log.TraceLevel, configuration.Level(), "default level is wrong")
	assert.Equal(t, os.Stderr, configuration.Writer(), "default output should be STDERR")
	assert.Empty(t, configuration.ContextExtractors(), "no default context extractors")
	excludeTime, format := configuration.TimeFieldConfiguration()
	assert.False(t, excludeTime, "by default add timestamp")
	assert.Equal(t, time.RFC3339, format, "default format should be time.RFC3339")
	includeCaller, skip := configuration.CallerConfiguration()
	assert.False(t, includeCaller, "by default don't include caller")
	assert.Equal(t, zerolog.CallerSkipFrameCount+skipWrapperFrames, skip, "skip frames by default is 4")
	assert.IsType(t, zerolog.Logger{}, configuration.Implementation(), "wrong implementation")
}

func TestCustomConfiguration(t *testing.T) {
	out := os.NewFile(0, os.DevNull)
	logger := Builder().
		SetWriter(out).
		SetLevel(log.WarnLevel).
		AddContextExtractors(
			func(ctx context.Context) map[string]interface{} { return nil },
		).
		ExcludeTime().SetCustomTimeFormatter("fake format").
		IncludeCallerAndSkipFrames(10).
		Build()
	configuration := logger.Configuration()
	assert.Equal(t, log.WarnLevel, configuration.Level(), "level is wrong")
	assert.Equal(t, out, configuration.Writer(), "writer is wrong")
	assert.Len(t, configuration.ContextExtractors(), 1, "there should be 1")
	excludeTime, format := configuration.TimeFieldConfiguration()
	assert.True(t, excludeTime, "exclude timestamp")
	assert.Equal(t, "fake format", format, "custom format is wrong")
	includeCaller, skip := configuration.CallerConfiguration()
	assert.True(t, includeCaller, "include caller")
	assert.Equal(t, zerolog.CallerSkipFrameCount+skipWrapperFrames+10, skip, "skip frames number is wrong")
	assert.IsType(t, zerolog.Logger{}, configuration.Implementation(), "wrong implementation")
}
func (s *wrapperTestSuite) SetupTest() {
	s.buf = &bytes.Buffer{}
	s.logger = Builder().SetWriter(s.buf).ExcludeTime().Build()
}

func TestSimpleOutputs(t *testing.T) {
	levelStrings := []string{"Trace", "debug", "Info", "warn", "ERROR"}
	for _, str := range levelStrings {
		buf := &bytes.Buffer{}
		logger := Builder().SetWriter(buf).ExcludeTime().Build()
		var logFunction func(context.Context, string, ...interface{})
		level := log.ParseLevel(str)
		switch level {
		case log.TraceLevel:
			logFunction = logger.Trace
		case log.DebugLevel:
			logFunction = logger.Debug
		case log.InfoLevel:
			logFunction = logger.Info
		case log.WarnLevel:
			logFunction = logger.Warn
		case log.ErrorLevel:
			logFunction = logger.Error
		default:
			assert.Fail(t, "unknown log level", "what is this %s", str)
		}
		// With Arguments
		logFunction(nil, "%s message", str)
		expected := fmt.Sprintf(`{"message":"%s message", "level":"%s"}`, str, level.String())
		assert.JSONEq(t, expected, buf.String())
		// Without Arguments
		buf.Reset()
		logFunction(nil, "message")
		expected = fmt.Sprintf(`{"message":"message", "level":"%s"}`, level.String())
		assert.JSONEq(t, expected, buf.String())
	}
}

func TestCustomLevel(t *testing.T) {
	levels := []log.Level{log.TraceLevel, log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel}
	for _, lvl := range levels {
		buf := &bytes.Buffer{}
		logger := Builder().SetWriter(buf).ExcludeTime().Build()
		logger.Custom(nil, lvl, "%s message", lvl.String())
		expected := fmt.Sprintf(`{"message":"%s message", "level":"%s"}`, lvl, lvl)
		assert.JSONEq(t, expected, buf.String())
		// Without Arguments
		buf.Reset()
		logger.Custom(nil, lvl, "message")
		expected = fmt.Sprintf(`{"message":"message", "level":"%s"}`, lvl)
		assert.JSONEq(t, expected, buf.String())
		// With Field/Error no Arguments
		buf.Reset()
		logger.WithField("field", "value").Custom(nil, lvl, "message")
		expected = fmt.Sprintf(`{"message":"message", "level":"%s", "field":"value"}`, lvl)
		assert.JSONEq(t, expected, buf.String())
		// With Field/Error with Arguments
		buf.Reset()
		logger.WithError(fmt.Errorf("bad one")).Custom(nil, lvl, "%s message", lvl)
		expected = fmt.Sprintf(`{"message":"%s message", "level":"%s", "error":"bad one"}`, lvl, lvl)
		assert.JSONEq(t, expected, buf.String())
	}
}

func TestTimeFieldInclusionAndFormat(t *testing.T) {
	var buf = &bytes.Buffer{}
	logger := Builder().SetWriter(buf).SetCustomTimeFormatter(time.RubyDate).Build()
	logger.Debug(nil, "Hello world")
	logOutput := buf.String()
	assert.Contains(t, logOutput, `"time":`)
	valuesMap := make(map[string]interface{})
	err := json.Unmarshal(buf.Bytes(), &valuesMap)
	assert.NoError(t, err, "unmarshal failed")
	timestamp, ok := valuesMap["time"].(string)
	assert.True(t, ok, "no such field 'time'")
	logTimestamp, _ := time.Parse(time.RubyDate, timestamp)
	assert.WithinDuration(t, time.Now(), logTimestamp, time.Second)
}

func TestCallerIncluded(t *testing.T) {
	var buf = &bytes.Buffer{}
	logger := Builder().SetWriter(buf).IncludeCallerAndSkipFrames(0).Build()
	// First log with fields - this changes the amount of frames to skip
	logger.
		WithField("one", 1).
		WithError(fmt.Errorf("this is an error")).
		WithField("two", 2).
		Debug(nil, "log with caller with fields")
	rx := `\"caller\"\:\".+\/go-masonry\/bzerolog\/wrapper_test\.go\:\d{1,5}` // we want to see this file as a caller
	assert.Regexp(t, rx, buf.String())
	buf.Reset()
	logger.Debug(nil, "log with caller no fields")
	assert.Regexp(t, rx, buf.String())
}

func TestStaticFieldsIncluded(t *testing.T) {
	var buf = &bytes.Buffer{}
	logger := Builder().SetWriter(buf).ExcludeTime().AddStaticFields(map[string]interface{}{
		"one": 1,
		"two": 2,
	}).Build()
	logger.
		WithError(fmt.Errorf("this is an error")).
		Info(nil, "log with fields")
	expected := `{"one": 1, "two": 2, "error": "this is an error", "message":"log with fields", "level":"info"}`
	assert.JSONEq(t, expected, buf.String())
}

func TestFieldsIncluded(t *testing.T) {
	var buf = &bytes.Buffer{}
	logger := Builder().SetWriter(buf).ExcludeTime().Build()
	logger.
		WithError(fmt.Errorf("this is an error")).
		WithField("one", 1).
		WithField("two", 2).
		Info(nil, "log with fields")
	expected := `{"one": 1, "two": 2, "error": "this is an error", "message":"log with fields", "level":"info"}`
	assert.JSONEq(t, expected, buf.String())
}

func TestContextExtractor(t *testing.T) {
	var buf = &bytes.Buffer{}
	logger := Builder().SetWriter(buf).ExcludeTime().AddContextExtractors(
		func(ctx context.Context) map[string]interface{} {
			value := ctx.Value("ctx1")
			return map[string]interface{}{
				"ctx1": value,
			}
		}, func(ctx context.Context) map[string]interface{} {
			value := ctx.Value("ctx2")
			return map[string]interface{}{
				"ctx2": value,
			}
		},
	).Build()
	withValue := context.WithValue(context.Background(), "ctx1", "one")
	ctx := context.WithValue(withValue, "ctx2", "two")
	logger.Info(ctx, "message with context extractors")
	expected := `{"ctx1": "one", "ctx2": "two", "level": "info", "message":"message with context extractors"}`
	assert.JSONEq(t, expected, buf.String())
}

func TestDefaultLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := Builder().SetWriter(buf).ExcludeTime().SetLevel(log.WarnLevel).Build()
	logger.Info(nil, "not to be seen")
	assert.Empty(t, buf.String())
}

func TestPanicExtractor(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := Builder().
		SetWriter(buf).
		ExcludeTime().
		AddContextExtractors(func(ctx context.Context) map[string]interface{} {
			panic("bad extractor")
		}).
		Build()
	logger.Info(context.Background(), "panic extractor")
	assert.Contains(t, buf.String(), "__panic__")
	assert.Contains(t, buf.String(), "bad extractor")
}

func TestConsoleWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := Builder().SetWriter(ConsoleWriter(buf)).ExcludeTime().Build()
	logger.Info(nil, "log message")
	assert.NotEmpty(t, buf.String())
}
