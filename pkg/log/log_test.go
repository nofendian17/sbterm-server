package log

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name  string
		opts  []Option
		run   func(l Logger)
		check func(t *testing.T, out string)
	}{
		{
			name: "level filtering drops lower levels",
			opts: []Option{WithLevel(LevelWarn)},
			run: func(l Logger) {
				l.Debug("debug-msg")
				l.Info("info-msg")
				l.Warn("warn-msg")
				l.Error("error-msg")
			},
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "warn-msg")
				assert.Contains(t, out, "error-msg")
				assert.NotContains(t, out, "debug-msg")
				assert.NotContains(t, out, "info-msg")
			},
		},
		{
			name: "json format produces valid json",
			opts: []Option{WithFormat(FormatJSON)},
			run: func(l Logger) {
				l.Info("json-msg", "count", 3)
			},
			check: func(t *testing.T, out string) {
				assert.True(t, json.Valid([]byte(out)), "expected valid JSON, got %q", out)
				assert.Contains(t, out, `"msg":"json-msg"`)
			},
		},
		{
			name: "with adds common attributes",
			run: func(l Logger) {
				l.With("service", "api").Info("with-msg")
			},
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "service=api")
			},
		},
		{
			name: "attributes via key value args",
			run: func(l Logger) {
				l.Info("args-msg", "count", 3)
			},
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "count=3")
			},
		},
		{
			name: "with group qualifies attributes",
			run: func(l Logger) {
				l.WithGroup("http").Info("group-msg", "method", "GET")
			},
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "http.method=GET")
			},
		},
		{
			name: "context variants log with context",
			run: func(l Logger) {
				l.DebugContext(context.Background(), "debug-ctx-msg")
				l.InfoContext(context.Background(), "info-ctx-msg")
				l.WarnContext(context.Background(), "warn-ctx-msg")
				l.ErrorContext(context.Background(), "error-ctx-msg")
			},
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "info-ctx-msg")
				assert.Contains(t, out, "warn-ctx-msg")
				assert.Contains(t, out, "error-ctx-msg")
				assert.NotContains(t, out, "debug-ctx-msg")
			},
		},
		{
			name: "default logger via SetDefault",
			run: func(l Logger) {
				SetDefault(l)
				Default().Info("default-msg")
			},
			check: func(t *testing.T, out string) {
				assert.Contains(t, out, "default-msg")
			},
		},
		{
			name: "enabled reflects configured level",
			run: func(l Logger) {
				assert.True(t, l.Enabled(context.Background(), LevelInfo), "expected LevelInfo to be enabled")
				assert.False(t, l.Enabled(context.Background(), LevelDebug), "expected LevelDebug to be disabled")
			},
			check: func(t *testing.T, out string) {
				assert.Empty(t, out, "expected no output")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := append(tt.opts, WithWriter(&buf))
			l := New(opts...)
			tt.run(l)

			if tt.check != nil {
				tt.check(t, buf.String())
			}
		})
	}
}

func TestSlogEscapeHatch(t *testing.T) {
	var buf bytes.Buffer
	l := New(WithWriter(&buf))

	s := l.Slog()
	require.NotNil(t, s, "Slog() returned nil")

	s.Info("raw-slog-msg")
	assert.Contains(t, buf.String(), "raw-slog-msg")
}
