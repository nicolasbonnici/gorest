package logger

import (
	"bytes"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func TestLogger_DefaultInitialization(t *testing.T) {
	if Log == nil {
		t.Fatal("Expected default logger to be initialized")
	}
}

func TestSetLogger(t *testing.T) {
	originalLogger := Log

	tests := []struct {
		name    string
		level   slog.Level
		handler slog.Handler
	}{
		{
			name:  "custom debug logger",
			level: slog.LevelDebug,
			handler: slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
		},
		{
			name:  "custom warn logger",
			level: slog.LevelWarn,
			handler: slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}),
		},
		{
			name:  "custom error logger",
			level: slog.LevelError,
			handler: slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelError,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customLogger := slog.New(tt.handler)
			SetLogger(customLogger)

			if Log != customLogger {
				t.Error("Expected logger to be replaced")
			}

			Log.Info("test message")
		})
	}

	SetLogger(originalLogger)
}

func TestSetLogger_Persistence(t *testing.T) {
	originalLogger := Log

	customLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	SetLogger(customLogger)

	if Log != customLogger {
		t.Error("Expected logger to persist after SetLogger call")
	}

	anotherLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	SetLogger(anotherLogger)

	if Log != anotherLogger {
		t.Error("Expected logger to be updated to new logger")
	}

	SetLogger(originalLogger)
}

func TestLogger_ThreadSafety(t *testing.T) {
	originalLogger := Log

	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			l := slog.New(slog.NewJSONHandler(io.Discard, nil))
			SetLogger(l)

			Log.Info("test message from goroutine", "id", id)
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}

	if Log == nil {
		t.Error("Logger should not be nil after concurrent access")
	}

	SetLogger(originalLogger)
}

func TestLogger_OutputFormat(t *testing.T) {
	originalLogger := Log

	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	customLogger := slog.New(handler)
	SetLogger(customLogger)

	Log.Info("test message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Error("Expected non-empty log output")
	}

	if !bytes.Contains([]byte(output), []byte("test message")) {
		t.Errorf("Expected log output to contain 'test message', got: %s", output)
	}

	if !bytes.Contains([]byte(output), []byte("key")) {
		t.Errorf("Expected log output to contain 'key', got: %s", output)
	}

	SetLogger(originalLogger)
}

func TestLogger_LevelFiltering(t *testing.T) {
	originalLogger := Log

	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	customLogger := slog.New(handler)
	SetLogger(customLogger)

	Log.Info("info message")
	infoOutput := buf.String()

	Log.Debug("debug message")
	debugOutput := buf.String()

	buf.Reset()

	Log.Warn("warn message")
	warnOutput := buf.String()

	if infoOutput != "" {
		t.Error("Expected info message to be filtered out at Warn level")
	}

	if debugOutput != "" {
		t.Error("Expected debug message to be filtered out at Warn level")
	}

	if warnOutput == "" {
		t.Error("Expected warn message to be logged at Warn level")
	}

	if !bytes.Contains([]byte(warnOutput), []byte("warn message")) {
		t.Errorf("Expected warn output to contain 'warn message', got: %s", warnOutput)
	}

	SetLogger(originalLogger)
}

func TestLogger_WithNilLogger(t *testing.T) {
	originalLogger := Log

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetLogger should not panic with nil logger: %v", r)
		}
		SetLogger(originalLogger)
	}()

	SetLogger(nil)

	if Log != nil {
		t.Error("Expected Log to be nil after SetLogger(nil)")
	}
}

func TestLogger_MultipleHandlers(t *testing.T) {
	originalLogger := Log

	var buf1, buf2 bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	jsonLogger := slog.New(jsonHandler)
	SetLogger(jsonLogger)
	Log.Info("json message")

	textHandler := slog.NewTextHandler(&buf2, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	textLogger := slog.New(textHandler)
	SetLogger(textLogger)
	Log.Info("text message")

	jsonOutput := buf1.String()
	textOutput := buf2.String()

	if !bytes.Contains([]byte(jsonOutput), []byte("json message")) {
		t.Errorf("Expected JSON output to contain message, got: %s", jsonOutput)
	}

	if !bytes.Contains([]byte(textOutput), []byte("text message")) {
		t.Errorf("Expected text output to contain message, got: %s", textOutput)
	}

	SetLogger(originalLogger)
}
