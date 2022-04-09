/*
 * Copyright 2021 The KunStack Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"go.uber.org/atomic"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	loggerKey = loggerKeyType("logger-key")
	// TraceLevel trace level
	TraceLevel Level = iota
	// DebugLevel when the log is set to Debug, all logs below this level will not be output
	DebugLevel
	// InfoLevel when the log is set to Info, all logs below this level will not be output
	InfoLevel
	// WarnLevel when the log is set to Warn, all logs below this level will not be output
	WarnLevel
	// ErrorLevel when the log is set to Error, all logs below this level will not be output
	ErrorLevel
	// PanicLevel when the log is set to Panic, all logs below this level will not be output
	PanicLevel
	// FatalLevel when the log is set to Fatal, all logs below this level will not be output
	FatalLevel
	// OffLevel when the log is set to off level, all levels of logs will not be output
	OffLevel
)

var (
	std atomic.Value
	_   Interface    = &Logger{}
	_   Interface    = &FiledLogger{}
	_   logr.LogSink = &sink{}
	_   io.Writer    = &safeWriter{}
)

func init() {
	std.Store(New(os.Stderr))
}

type (
	loggerKeyType string
	// Level Used to indicate the log level
	Level int32
	// Option Log options
	Option func(l *Logger)
	// TimeEncoder 用来对时间进行格式化
	TimeEncoder func(t *time.Time) string
	// CallerEncoder Used to format the call stack
	CallerEncoder func(file string, line int) string
)

// safeWriter thread-safe writer
type safeWriter struct {
	out    io.Writer
	locker sync.Mutex
}

func (w *safeWriter) Write(p []byte) (n int, err error) {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.out.Write(p)
}

// Interface logger with field
type Interface interface {
	WithFields(fields ...interface{}) Interface
	WithError(err error) Interface
	WithOutput(out io.Writer) Interface
	WithLevel(level Level) Interface
	WithEncoder(Encoder) Interface
	WithTimeEncoder(TimeEncoder) Interface
	WithMaxVerbosity(verbosity int) Interface
	WithCallerEncoder(CallerEncoder) Interface
	V(v int) Interface
	Verbosity() int
	Output(v int, level Level, callDepth int, fields Fields, s string) error
	WithContext(ctx context.Context, fields ...interface{}) context.Context
	Fields() Fields
	Sink() logr.LogSink
	Printf(format string, v ...interface{})
	Print(v ...interface{})
	Println(v ...interface{})
	Trace(v ...interface{})
	Tracef(format string, v ...interface{})
	Traceln(v ...interface{})
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Debugln(v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Infoln(v ...interface{})
	Warning(v ...interface{})
	Warningf(format string, v ...interface{})
	Warningln(v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Errorln(v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
	Panicln(v ...interface{})
}

func fieldMap(kv ...interface{}) Fields {
	fields := make(Fields, len(kv)/2)

	for i, n := 0, len(kv); i < n; i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			k = fmt.Sprintf("!(%#v)", kv[i])
		}
		var v string
		if i+1 < n {
			v = fmt.Sprintf("%+v", kv[i+1])
		}
		fields[k] = v
	}
	return fields
}

// TimeRFC3339 RFC3339 time encoder
func TimeRFC3339() TimeEncoder {
	return func(t *time.Time) string {
		return t.Format(time.RFC3339)
	}
}

// TimeRFC850 RFC850 time encoder
func TimeRFC850() TimeEncoder {
	return func(t *time.Time) string {
		return t.Format(time.RFC850)
	}
}

// TimeRFC1123 RFC1123 time encoder
func TimeRFC1123() TimeEncoder {
	return func(t *time.Time) string {
		return t.Format(time.RFC850)
	}
}

// TimeRFC822  RFC822 time encoder
func TimeRFC822() TimeEncoder {
	return func(t *time.Time) string {
		return t.Format(time.RFC822)
	}
}

// TimeStamp Stamp time encoder
func TimeStamp() TimeEncoder {
	return func(t *time.Time) string {
		return t.Format(time.Stamp)
	}
}

// TimeRFC3339Nano RFC3339Nano time encoder
func TimeRFC3339Nano() TimeEncoder {
	return func(t *time.Time) string {
		return t.Format(time.RFC3339Nano)
	}
}

// ShortCaller short caller encoder
func ShortCaller() CallerEncoder {
	return func(file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		return short + ":" + strconv.Itoa(line)
	}
}

// Fields Store key-value data of type string
type Fields map[string]string

// Get value from Fields
func (f Fields) Get(k string) interface{} {
	v, ok := f[k]
	if !ok {
		return ""
	}
	return v
}

// Set value to Fields
func (f Fields) Set(k string, v string) { f[k] = v }

// Logger High-performance log structure
// We are willing to accept any method that can increase the speed
// If consuming memory can increase the speed, then just do it
type Logger struct {
	verbosity     int
	v             int
	level         Level
	bufferSize    int
	encoder       Encoder
	timeEncoder   TimeEncoder
	callerEncoder CallerEncoder
	writer        *safeWriter
}

func (l *Logger) V(v int) Interface {
	c := *l
	c.v = v
	return &c
}

func (l *Logger) WithMaxVerbosity(verbosity int) Interface {
	c := *l
	c.verbosity = verbosity
	return &c
}

func (l *Logger) Verbosity() int {
	return l.v
}

// FiledLogger logger but with field
type FiledLogger struct {
	l      Interface
	fields Fields // never need lock
}

func (l *FiledLogger) Verbosity() int {
	return l.l.Verbosity()
}

func (l *FiledLogger) WithMaxVerbosity(verbosity int) Interface {
	return &FiledLogger{l: l.l.WithMaxVerbosity(verbosity), fields: l.fields}
}

func (l *FiledLogger) V(v int) Interface {
	return &FiledLogger{l: l.l.V(v), fields: l.fields}
}

func (l *Logger) Sink() logr.LogSink {
	return &sink{
		logger: l,
	}
}

func (l *FiledLogger) Sink() logr.LogSink {
	return &sink{
		logger: l,
	}
}

// WithFields creates an FiledLogger from the  logger and adds multiple
// fields to it. This is simply a helper for `WithFields`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the FiledLogger it returns.
func (l *FiledLogger) WithFields(fields ...interface{}) Interface {
	data := make(Fields, len(l.fields)+len(fields)/2)

	vars := fieldMap(fields...)

	for k, v := range vars {
		data[k] = v
	}

	for k, v := range l.fields {
		data[k] = v
	}

	return &FiledLogger{l: l.l, fields: data}
}

// WithError creates an FiledLogger from the logger and adds an error to it, using the value defined in error as key.
func (l *FiledLogger) WithError(err error) Interface {
	if err == nil {
		return l
	}
	return l.WithFields(Fields{"error": err.Error()})
}

// WithOutput create new logger and set the output
func (l *FiledLogger) WithOutput(out io.Writer) Interface {
	return &FiledLogger{l: l.l.WithOutput(out), fields: l.fields}
}

// WithLevel Set the minimum log level that can be displayed.
// Logs below this level will not be displayed.
func (l *FiledLogger) WithLevel(level Level) Interface {
	return &FiledLogger{l: l.l.WithLevel(level), fields: l.fields}
}

// WithEncoder set encoder for logging
func (l *FiledLogger) WithEncoder(encoder Encoder) Interface {
	return &FiledLogger{l: l.l.WithEncoder(encoder), fields: l.fields}
}

// WithTimeEncoder set time encoder for logger
func (l *FiledLogger) WithTimeEncoder(encoder TimeEncoder) Interface {
	return &FiledLogger{l: l.l.WithTimeEncoder(encoder), fields: l.fields}
}

// WithCallerEncoder set caller encoder for logger
func (l *FiledLogger) WithCallerEncoder(encoder CallerEncoder) Interface {
	return &FiledLogger{l: l.l.WithCallerEncoder(encoder), fields: l.fields}
}

// Output fake output is just call l.l.Output
func (l *FiledLogger) Output(v int, level Level, callDepth int, fields Fields, s string) error {
	return l.l.Output(v, level, callDepth+1, fields, s)
}

// WithContext Call l.WithFields (fields) to create a logger and inject it into the context
func (l *FiledLogger) WithContext(ctx context.Context, fields ...interface{}) context.Context {
	return context.WithValue(ctx, loggerKey, l.WithFields(fieldMap(fields...)))
}

// Fields  get fields
func (l *FiledLogger) Fields() Fields {
	f := make(Fields, len(l.fields))
	for k, v := range l.fields {
		f[k] = v
	}
	return f
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Printf(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), InfoLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *FiledLogger) Print(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), InfoLevel, 2, l.fields, fmt.Sprint(v...))
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *FiledLogger) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), InfoLevel, 2, l.fields, s[:len(s)-1])
}

// Trace calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Trace(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), TraceLevel, 2, l.fields, fmt.Sprint(v...))
}

// Tracef calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Tracef(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), TraceLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Traceln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Traceln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), TraceLevel, 2, l.fields, s[:len(s)-1])
}

// Debug calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Debug(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), DebugLevel, 2, l.fields, fmt.Sprint(v...))
}

// Debugf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Debugf(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), DebugLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Debugln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Debugln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), DebugLevel, 2, l.fields, s[:len(s)-1])
}

// Info calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Info(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), InfoLevel, 2, l.fields, fmt.Sprint(v...))
}

// Infof calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Infof(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), InfoLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Infoln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Infoln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), InfoLevel, 2, l.fields, s[:len(s)-1])
}

// Warning calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warning(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), WarnLevel, 2, l.fields, fmt.Sprint(v...))
}

// Warningf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warningf(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), WarnLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Warningln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warningln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), WarnLevel, 2, l.fields, s[:len(s)-1])
}

// Error calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Error(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), ErrorLevel, 2, l.fields, fmt.Sprint(v...))
}

// Errorf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Errorf(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), ErrorLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Errorln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), ErrorLevel, 2, l.fields, s[:len(s)-1])
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *FiledLogger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = l.Output(l.l.Verbosity(), PanicLevel, 2, l.fields, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *FiledLogger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = l.Output(l.l.Verbosity(), PanicLevel, 2, l.fields, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *FiledLogger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), PanicLevel, 2, l.fields, s[:len(s)-1])
	panic(s)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *FiledLogger) Fatal(v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), FatalLevel, 2, l.fields, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *FiledLogger) Fatalf(format string, v ...interface{}) {
	_ = l.Output(l.l.Verbosity(), FatalLevel, 2, l.fields, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *FiledLogger) Fatalln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.l.Verbosity(), FatalLevel, 2, l.fields, s[:len(s)-1])
	os.Exit(1)
}

// ParseLevel parse log level from plain text
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "", "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "off":
		return OffLevel, nil
	}
	return 0, fmt.Errorf("not support log level %s", lvl)
}

// ParseEncoder parse log Formatter from plain text
func ParseEncoder(f string) (Encoder, error) {
	switch strings.ToLower(f) {
	case "", "text":
		return DefaultTextEncoder, nil
	case "json":
		return DefaultJSONEncoder, nil
	}
	return nil, fmt.Errorf("not support formatter %s", f)
}

// ParseTimeEncoder parse time encoder from plain text
func ParseTimeEncoder(f string) (TimeEncoder, error) {
	switch strings.ToUpper(f) {
	case "":
		return nil, nil
	case "RFC3339":
		return TimeRFC3339(), nil
	case "RFC3339NANO":
		return TimeRFC3339Nano(), nil
	case "RFC822":
		return TimeRFC822(), nil
	case "RFC850":
		return TimeRFC850(), nil
	case "RFC1123":
		return TimeRFC1123(), nil
	case "STAMP":
		return TimeStamp(), nil
	}
	return nil, fmt.Errorf("unknown time encoder %s", f)
}

// ParseCaller parse log caller from plain text
func ParseCaller(f string) (CallerEncoder, error) {
	switch strings.ToLower(f) {
	case "", "none":
		return nil, nil
	case "short":
		return ShortCaller(), nil
	case "long", "full":
		return FullCaller(), nil
	}
	return nil, fmt.Errorf("not support caller %s", f)
}

// AddLevel wrap the level as an option
func AddLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// AddMaxVerbosity wrap the verbosity as an option
func AddMaxVerbosity(verbosity int) Option {
	return func(l *Logger) {
		l.verbosity = verbosity
	}
}

// AddEncoder  wrap the encoder as an option
func AddEncoder(enc Encoder) Option {
	return func(l *Logger) {
		l.encoder = enc
	}
}

// AddTimeEncoder wrap the time encoder as an option
func AddTimeEncoder(enc TimeEncoder) Option {
	return func(l *Logger) {
		l.timeEncoder = enc
	}
}

// AddCallerEncoder wrap the caller encoder as an option
func AddCallerEncoder(enc CallerEncoder) Option {
	return func(l *Logger) {
		l.callerEncoder = enc
	}
}

// AddBufferSize wrap the caller encoder as an option
func AddBufferSize(size int) Option {
	return func(l *Logger) {
		l.bufferSize = size
	}
}

// AddTimeRFC3339 Return an Option This option will
// set the logger's time decoder to RFC3339
func AddTimeRFC3339() Option {
	return AddTimeEncoder(TimeRFC3339())
}

// AddTimeRFC850 Return an Option This option will
// set the logger's time decoder to RFC850
func AddTimeRFC850() Option {
	return AddTimeEncoder(TimeRFC850())
}

// AddTimeRFC1123 Return an Option This option will
// set the logger's time decoder to RFC1123
func AddTimeRFC1123() Option {
	return AddTimeEncoder(TimeRFC1123())
}

// AddTimeRFC822 Return an Option This option will
// set the logger's time decoder to RFC822
func AddTimeRFC822() Option {
	return AddTimeEncoder(TimeRFC822())
}

// AddTimeStamp Return an Option This option will
// set the logger's time decoder to Stamp
func AddTimeStamp() Option {
	return AddTimeEncoder(TimeStamp())
}

// AddTimeRFC3339Nano Return an Option This option will
// set the logger's time decoder to RFC3339Nano
func AddTimeRFC3339Nano() Option {
	return AddTimeEncoder(TimeRFC3339Nano())
}

// AddJSONEncoder Return an Option This option will
// set the logger's format decoder to json
func AddJSONEncoder() Option {
	return AddEncoder(DefaultJSONEncoder)
}

// AddTextEncoder Return an Option This option will
// set the logger's format decoder to text
func AddTextEncoder() Option {
	return AddEncoder(DefaultTextEncoder)
}

// AddFullCaller Return an Option This option will
// set the logger's caller decoder to FullCaller
func AddFullCaller() Option {
	return AddCallerEncoder(FullCaller())
}

// AddShortCaller Return an Option This option will
// set the logger's caller decoder to ShortCaller
func AddShortCaller() Option {
	return AddCallerEncoder(ShortCaller())
}

// FullCaller full caller encoder
func FullCaller() CallerEncoder {
	return func(file string, line int) string {
		return file + ":" + strconv.Itoa(line)
	}
}

// StrField Shortcut, return func(f Fields), with key,value
func StrField(key string, value ...string) func(f Fields) {
	return func(f Fields) {
		f[key] = strings.Join(value, ",")
	}
}

// IntField Shortcut, return func(f Fields), with key,value
func IntField(key string, value ...int) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, strconv.Itoa(v))
		}
		f[key] = strings.Join(values, ",")
	}
}

// Int32Field Shortcut, return func(f Fields), with key,value
func Int32Field(key string, value ...int32) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, strconv.Itoa(int(v)))
		}
		f[key] = strings.Join(values, ",")
	}
}

// Int64Field Shortcut, return func(f Fields), with key,value
func Int64Field(key string, value ...int64) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, strconv.FormatInt(v, 10))
		}
		f[key] = strings.Join(values, ",")
	}
}

// Float64Field Shortcut, return func(f Fields), with key,value
func Float64Field(key string, value ...float64) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, fmt.Sprintf("%f", v))
		}
		f[key] = strings.Join(values, ",")
	}
}

// Float32Field Shortcut, return func(f Fields), with key,value
func Float32Field(key string, value ...float32) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, fmt.Sprintf("%f", v))
		}
		f[key] = strings.Join(values, ",")
	}
}

// BoolField Shortcut, return func(f Fields), with key(type string),value(type bool)
func BoolField(key string, value ...bool) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, strconv.FormatBool(v))
		}
		f[key] = strings.Join(values, ",")
	}
}

// DurationField Shortcut, return func(f Fields), with key(type string),value(type time.Duration)
func DurationField(key string, value ...time.Duration) func(f Fields) {
	return func(f Fields) {
		values := make([]string, 0, len(value))
		for _, v := range value {
			values = append(values, v.String())
		}
		f[key] = strings.Join(values, ",")
	}
}

// WithoutContext just return std logger
func WithoutContext() Interface {
	return std.Load().(Interface)
}

// WithContext Inject logger into the context
func WithContext(ctx context.Context, fields ...interface{}) context.Context {
	std := std.Load().(Interface)
	return std.WithContext(ctx, fields...)
}

// FromContext get logger from context
func FromContext(ctx context.Context) Interface {
	if ctx == nil {
		panic("nil context")
	}
	logger, ok := ctx.Value(loggerKey).(Interface)
	if !ok {
		logger = std.Load().(Interface)
	}
	return logger
}

// New create Interface and set output is w
func New(w io.Writer, opts ...Option) Interface {
	l := &Logger{
		level:   InfoLevel,
		encoder: DefaultTextEncoder,
		writer:  &safeWriter{out: w},
	}

	// apply option
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// WithContext Inject logger into context
func (l *Logger) WithContext(ctx context.Context, fields ...interface{}) context.Context {
	return context.WithValue(ctx, loggerKey, l.WithFields(fields...))
}

// WithLevel Set the minimum log level that can be displayed.
// Logs below this level will not be displayed.
func (l *Logger) WithLevel(level Level) Interface {
	c := *l
	c.level = level
	return &c
}

// WithEncoder set encoder for logging
func (l *Logger) WithEncoder(encoder Encoder) Interface {
	c := *l
	c.encoder = encoder
	return &c
}

// WithTimeEncoder set time encoder for loger
func (l *Logger) WithTimeEncoder(encoder TimeEncoder) Interface {
	c := *l
	c.timeEncoder = encoder
	return &c
}

// WithCallerEncoder set caller encoder for logger
func (l *Logger) WithCallerEncoder(encoder CallerEncoder) Interface {
	c := *l
	c.callerEncoder = encoder
	return &c
}

// Output call the format function and write the formatted data to io.Writer
func (l *Logger) Output(v int, level Level, callDepth int, fields Fields, s string) error {
	var (
		caller   string
		dateTime string
	)

	// skip output by level/verbosity limit
	if l.level > level || l.verbosity < v {
		return nil
	}

	if l.timeEncoder != nil {
		t := time.Now() // Sacrificing time accuracy, but gaining performance
		dateTime = l.timeEncoder(&t)
	}

	if l.callerEncoder != nil {
		_, file, line, ok := runtime.Caller(callDepth) // Expensive time consumption
		if !ok {
			file = "???"
			line = 0
		}
		caller = l.callerEncoder(file, line)
	}

	buf := bufPool.Get()
	defer bufPool.Put(buf)

	if err := l.encoder.Encode(buf, &Message{
		Level:   level,
		Time:    dateTime,
		Caller:  caller,
		Fields:  fields,
		Message: s,
	}); err != nil {
		return err
	}
	_, err := l.writer.Write(*buf)
	return err
}

// Fields  get fields
func (l *Logger) Fields() Fields { return nil }

// WithOutput set the Logger writer to out io.Writer
func (l *Logger) WithOutput(out io.Writer) Interface {
	c := *l
	c.writer = &safeWriter{out: out}
	return &c
}

// WithError key is string value is Error,
// this key value will be printed out every time when log is printed
func (l *Logger) WithError(err error) Interface {
	return l.WithFields(Fields{"error": err.Error()})
}

// WithFields Parameter is map[string] string type
// this key value will be printed out every time when log is printed
func (l *Logger) WithFields(fields ...interface{}) Interface {
	return &FiledLogger{
		l:      l,
		fields: fieldMap(fields...),
	}
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	_ = l.Output(l.v, InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) {
	_ = l.Output(l.v, InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, InfoLevel, 2, nil, s[:len(s)-1])
}

// Trace calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Trace(v ...interface{}) {
	_ = l.Output(l.v, TraceLevel, 2, nil, fmt.Sprint(v...))
}

// Tracef calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Tracef(format string, v ...interface{}) {
	_ = l.Output(l.v, TraceLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Traceln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Traceln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, TraceLevel, 2, nil, s[:len(s)-1])
}

// Debug calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debug(v ...interface{}) {
	_ = l.Output(l.v, DebugLevel, 2, nil, fmt.Sprint(v...))
}

// Debugf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debugf(format string, v ...interface{}) {
	_ = l.Output(l.v, DebugLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Debugln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debugln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, DebugLevel, 2, nil, s[:len(s)-1])
}

// Info calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Info(v ...interface{}) {
	_ = l.Output(l.v, InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Infof calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infof(format string, v ...interface{}) {
	_ = l.Output(l.v, InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Infoln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infoln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, InfoLevel, 2, nil, s[:len(s)-1])
}

// Warning calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warning(v ...interface{}) {
	_ = l.Output(l.v, WarnLevel, 2, nil, fmt.Sprint(v...))
}

// Warningf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningf(format string, v ...interface{}) {
	_ = l.Output(l.v, WarnLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Warningln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, WarnLevel, 2, nil, s[:len(s)-1])
}

// Error calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Error(v ...interface{}) {
	_ = l.Output(l.v, ErrorLevel, 2, nil, fmt.Sprint(v...))
}

// Errorf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorf(format string, v ...interface{}) {
	_ = l.Output(l.v, ErrorLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Errorln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, ErrorLevel, 2, nil, s[:len(s)-1])
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = l.Output(l.v, PanicLevel, 2, nil, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = l.Output(l.v, PanicLevel, 2, nil, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, PanicLevel, 2, nil, s[:len(s)-1])
	panic(s)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	_ = l.Output(l.v, FatalLevel, 2, nil, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	_ = l.Output(l.v, FatalLevel, 2, nil, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(l.v, FatalLevel, 2, nil, s[:len(s)-1])
	os.Exit(1)
}

type sink struct {
	callDepth int
	logger    Interface
}

func (l *sink) Init(info logr.RuntimeInfo) {
	l.callDepth += info.CallDepth
}

func (l *sink) Enabled(level int) bool {
	return l.logger.Verbosity() >= level
}

func (l *sink) Info(level int, msg string, keysAndValues ...interface{}) {
	_ = l.logger.Output(level, InfoLevel, l.callDepth, l.logger.WithFields(fieldMap(keysAndValues...)).Fields(), msg)
}

func (l *sink) Error(err error, msg string, keysAndValues ...interface{}) {
	_ = l.logger.Output(l.logger.Verbosity(), ErrorLevel, l.callDepth, l.logger.WithError(err).WithFields(fieldMap(keysAndValues...)).Fields(), msg)
}

func (l *sink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	fmt.Printf("%+v----\n", keysAndValues)
	fields := make([]interface{}, 0, len(keysAndValues)*2)

	for i := 0; i < len(keysAndValues); i++ {
		vars := keysAndValues[i].([]interface{})
		fields = append(fields, vars...)
	}

	return &sink{
		callDepth: l.callDepth,
		logger:    l.logger.WithFields(fields),
	}
}

func (l *sink) WithName(name string) logr.LogSink {
	return &sink{
		callDepth: l.callDepth,
		logger:    l.logger.WithFields("name", name),
	}
}

// WithOutput set output to out
func WithOutput(out io.Writer) Interface {
	std := std.Load().(Interface)
	return std.WithOutput(out)
}

// WithEncoder set Encoder to e for std logger
func WithEncoder(e Encoder) Interface {
	std := std.Load().(Interface)
	return std.WithEncoder(e)
}

// WithTimeEncoder set time encoder for std logger
func WithTimeEncoder(e TimeEncoder) Interface {
	std := std.Load().(Interface)
	return std.WithTimeEncoder(e)
}

// WithCallerEncoder set caller encoder for std logger
func WithCallerEncoder(e CallerEncoder) Interface {
	std := std.Load().(Interface)
	return std.WithCallerEncoder(e)
}

// WithFields Parameter is map[string] string type
// this key value will be printed out every time when log is printed
func WithFields(fields ...interface{}) Interface {
	std := std.Load().(Interface)
	return std.WithFields(fields)
}

// WithError key is string value is Error,
// this key value will be printed out every time when log is printed
func WithError(err error) Interface {
	std := std.Load().(Interface)
	return std.WithError(err)
}

// WithLevel Set the log level, logs below this level will not be printed
func WithLevel(level Level) Interface {
	std := std.Load().(Interface)
	return std.WithLevel(level)
}

// Printf calls std.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Print calls std.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Println calls std.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), InfoLevel, 2, nil, s[:len(s)-1])
}

// Trace calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Trace(v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), TraceLevel, 2, nil, fmt.Sprint(v...))
}

// Tracef calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Tracef(format string, v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), TraceLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Traceln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Traceln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), TraceLevel, 2, nil, s[:len(s)-1])
}

// Debug calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Debug(v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), DebugLevel, 2, nil, fmt.Sprint(v...))
}

// Debugf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), DebugLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Debugln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Debugln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), DebugLevel, 2, nil, s[:len(s)-1])
}

// Info calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Info(v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Infof calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Infoln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Infoln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), InfoLevel, 2, nil, s[:len(s)-1])
}

// Error calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Error(v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), ErrorLevel, 2, nil, fmt.Sprint(v...))
}

// Errorf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), ErrorLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Errorln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), ErrorLevel, 2, nil, s[:len(s)-1])
}

// Panic is equivalent to l.Print() followed by a call to panic().
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), PanicLevel, 2, nil, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), PanicLevel, 2, nil, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), PanicLevel, 2, nil, s[:len(s)-1])
	panic(s)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), FatalLevel, 2, nil, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), FatalLevel, 2, nil, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func Fatalln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std := std.Load().(Interface)
	_ = std.Output(std.Verbosity(), FatalLevel, 2, nil, s[:len(s)-1])
	os.Exit(1)
}

// Sink is equivalent to l.Sink()
func Sink() logr.LogSink {
	std := std.Load().(Interface)
	return std.Sink()
}

// SetLogger .
func SetLogger(l Interface) {
	std.Store(l)
}

// NewFromOptions create logger from options
func NewFromOptions(opts *Options) (Interface, error) {
	logger := New(os.Stderr)

	lvl, err := ParseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	logger = logger.WithLevel(lvl)

	encoder, err := ParseEncoder(opts.Format)
	if err != nil {
		return nil, err
	}
	logger = logger.WithEncoder(encoder)

	cal, err := ParseCaller(opts.Caller)
	if err != nil {
		return nil, err
	}
	logger = logger.WithCallerEncoder(cal)

	timeEncoder, err := ParseTimeEncoder(opts.Time)
	if err != nil {
		return nil, err
	}
	logger = logger.WithTimeEncoder(timeEncoder)

	if opts.File != "" {
		file, err := os.OpenFile(opts.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		logger = logger.WithOutput(file)
	}

	return logger.WithMaxVerbosity(opts.MaxVerbosity), nil
}
