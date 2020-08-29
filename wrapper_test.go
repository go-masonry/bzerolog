package bzerolog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-masonry/mortar/interfaces/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfiguration(t *testing.T) {
	logger := Builder().Build()
	configuration := logger.Configuration()
	assert.Equal(t, log.TraceLevel, configuration.Level(), "default level is wrong")
	assert.IsType(t, zerolog.Logger{}, configuration.Implementation(), "wrong implementation")
}

func TestCustomConfiguration(t *testing.T) {
	logger := Builder().
		SetLevel(log.WarnLevel).
		IncrementSkipFrames(10).
		Build()
	assert.Equal(t, log.WarnLevel, logger.Configuration().Level(), "level is wrong")
	assert.IsType(t, zerolog.Logger{}, logger.Configuration().Implementation(), "wrong implementation")
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
			t.Fatalf("unknown log level %s", str)
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
		logger.Custom(nil, lvl, 0, "%s message", lvl.String())
		expected := fmt.Sprintf(`{"message":"%s message", "level":"%s"}`, lvl, lvl)
		assert.JSONEq(t, expected, buf.String())
		// Without Arguments
		buf.Reset()
		logger.Custom(nil, lvl, 0, "message")
		expected = fmt.Sprintf(`{"message":"message", "level":"%s"}`, lvl)
		assert.JSONEq(t, expected, buf.String())
		// With Field/Error no Arguments
		buf.Reset()
		logger.WithField("field", "value").Custom(nil, lvl, 0, "message")
		expected = fmt.Sprintf(`{"message":"message", "level":"%s", "field":"value"}`, lvl)
		assert.JSONEq(t, expected, buf.String())
		// With Field/Error with Arguments
		buf.Reset()
		logger.WithError(fmt.Errorf("bad one")).Custom(nil, lvl, 0, "%s message", lvl)
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
	logger := Builder().SetWriter(buf).IncludeCaller().Build()
	// First log with fields - this changes the amount of frames to skip
	logger.
		WithField("one", 1).
		WithError(fmt.Errorf("this is an error")).
		WithField("two", 2).
		Debug(nil, "log with caller with fields")
	rx := `caller":".+bzerolog` // we want to see this file as a caller
	assert.Regexp(t, rx, buf.String())
	buf.Reset()
	logger.Debug(nil, "log with caller no fields")
	assert.Regexp(t, rx, buf.String())
}

func TestCallerIncludedCustom(t *testing.T) {
	var buf = &bytes.Buffer{}
	logger := Builder().SetWriter(buf).IncludeCaller().Build()
	// First log with fields - this changes the amount of frames to skip
	logger.
		WithField("one", 1).
		WithError(fmt.Errorf("this is an error")).
		WithField("two", 2).
		Custom(nil, log.DebugLevel, 0, "log with caller and fields")
	rx := `caller":".+bzerolog` // we want to see this file as a caller
	assert.Regexp(t, rx, buf.String())
	buf.Reset()
	logger.Custom(nil, log.DebugLevel, 0, "log with caller no fields")
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

func TestDefaultLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := Builder().SetWriter(buf).ExcludeTime().SetLevel(log.WarnLevel).Build()
	logger.Info(nil, "not to be seen")
	assert.Empty(t, buf.String())
}

func TestConsoleWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := Builder().SetWriter(ConsoleWriter(buf)).ExcludeTime().Build()
	logger.Info(nil, "log message")
	assert.NotEmpty(t, buf.String())
}
