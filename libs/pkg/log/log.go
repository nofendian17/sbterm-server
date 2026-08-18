// Package log provides a thin, source-accurate wrapper around log/slog.
//
// The wrapper re-implements slog's "Wrapping output methods" pattern
// (runtime.Callers + slog.NewRecord) so that log records report the caller's
// file and line instead of this package's internals.
// Source: https://pkg.go.dev/log/slog#hdr-Wrapping_output_methods
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type Logger interface {
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	With(args ...any) Logger
	WithGroup(name string) Logger
	Enabled(ctx context.Context, level Level) bool
	Slog() *slog.Logger
}

type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Option func(*options)

type options struct {
	level     Level
	format    Format
	writer    io.Writer
	addSource bool
}

func WithLevel(level Level) Option {
	return func(o *options) { o.level = level }
}

func WithFormat(format Format) Option {
	return func(o *options) { o.format = format }
}

func WithWriter(w io.Writer) Option {
	return func(o *options) { o.writer = w }
}

func WithAddSource(enabled bool) Option {
	return func(o *options) { o.addSource = enabled }
}

type slogger struct {
	l *slog.Logger
}

func New(opts ...Option) Logger {
	o := &options{
		level:  LevelInfo,
		format: FormatText,
		writer: os.Stdout,
	}
	for _, opt := range opts {
		opt(o)
	}

	var h slog.Handler
	hopts := &slog.HandlerOptions{Level: o.level, AddSource: o.addSource}
	if o.format == FormatJSON {
		h = slog.NewJSONHandler(o.writer, hopts)
	} else {
		h = slog.NewTextHandler(o.writer, hopts)
	}

	return &slogger{l: slog.New(h)}
}

var defaultLogger Logger = New()

func Default() Logger {
	return defaultLogger
}

func SetDefault(l Logger) {
	defaultLogger = l
}

func (s *slogger) Slog() *slog.Logger {
	return s.l
}

func (s *slogger) With(args ...any) Logger {
	return &slogger{l: s.l.With(args...)}
}

func (s *slogger) WithGroup(name string) Logger {
	return &slogger{l: s.l.WithGroup(name)}
}

func (s *slogger) Enabled(ctx context.Context, level Level) bool {
	return s.l.Enabled(ctx, level)
}

func (s *slogger) Debug(msg string, args ...any) {
	s.log(context.Background(), LevelDebug, msg, args...)
}
func (s *slogger) Info(msg string, args ...any) { s.log(context.Background(), LevelInfo, msg, args...) }
func (s *slogger) Warn(msg string, args ...any) { s.log(context.Background(), LevelWarn, msg, args...) }
func (s *slogger) Error(msg string, args ...any) {
	s.log(context.Background(), LevelError, msg, args...)
}

func (s *slogger) DebugContext(ctx context.Context, msg string, args ...any) {
	s.log(ctx, LevelDebug, msg, args...)
}

func (s *slogger) InfoContext(ctx context.Context, msg string, args ...any) {
	s.log(ctx, LevelInfo, msg, args...)
}

func (s *slogger) WarnContext(ctx context.Context, msg string, args ...any) {
	s.log(ctx, LevelWarn, msg, args...)
}

func (s *slogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	s.log(ctx, LevelError, msg, args...)
}

func (s *slogger) log(ctx context.Context, level Level, msg string, args ...any) {
	if !s.l.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip: runtime.Callers, log, exported method
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = s.l.Handler().Handle(ctx, r)
}
