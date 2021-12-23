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
	"bufio"
	"context"
	"fmt"
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
	TraceLevel Level = iota << 1
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
	// the size of the default buffer 4k
	defaultBufSize = 4096
	//flush the buffer interval
	defaultFlushInterval = time.Second * 5
)

var (
	std Interface = New(os.Stderr)
	_   Interface = &Logger{}
	_   Interface = &FiledLogger{}
)

type (
	loggerKeyType string
	// Level Used to indicate the log level
	Level int32
	// TimeEncoder 用来对时间进行格式化
	TimeEncoder func(t *time.Time) string
	// CallerEncoder Used to format the call stack
	CallerEncoder func(file string, line int) string
	// Option Log options
	Option func(l *config)
)

// Interface logger with field
type Interface interface {
	WithField(key string, value ...string) Interface
	WithFields(fields Fields) Interface
	WithError(err error) Interface
	WithFloat64Field(key string, value ...float64) Interface
	WithFloat32Field(key string, value ...float32) Interface
	WithInt64Field(key string, value ...int64) Interface
	WithIntField(key string, value ...int) Interface
	WithBoolField(key string, value ...bool) Interface
	WithDurationField(key string, value ...time.Duration) Interface
	SetOutput(out io.Writer)
	SetLevel(level Level)
	SetEncoder(Encoder)
	SetTimeEncoder(TimeEncoder)
	SetCallerEncoder(CallerEncoder)
	Output(level Level, calldepth int, fileds Fields, s string) error
	WithContext(ctx context.Context, opts ...func(Fields)) context.Context
	Fields() Fields
	Flush()
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
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	Warnln(v ...interface{})
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
func (f Fields) Get(k string) string {
	v, ok := f[k]
	if !ok {
		return ""
	}
	return v
}

// Set value to Fields
func (f Fields) Set(k, v string) { f[k] = v }

// config Parameter structure required for log initialization
type config struct {
	level         Level
	bufferSize    int
	encoder       Encoder
	timeEncoder   TimeEncoder
	callerEncoder CallerEncoder
	flushInterval time.Duration
}

// Logger High-performance log structure
// We are willing to accept any method that can increase the speed
// If consuming memory can increase the speed, then just do it
type Logger struct {
	*config
	locker      sync.Mutex
	stopFlush   chan struct{}
	bufioWriter *bufio.Writer
}

// FiledLogger logger but with field
type FiledLogger struct {
	l      *Logger
	fields Fields // never need lock
}

// WithField creates an FiledLogger from the  logger and adds multiple
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the FiledLogger it returns.
func (l *FiledLogger) WithField(key string, value ...string) Interface {
	return l.WithFields(Fields{key: strings.Join(value, ",")})
}

// WithFields creates an FiledLogger from the  logger and adds multiple
// fields to it. This is simply a helper for `WithFields`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the FiledLogger it returns.
func (l *FiledLogger) WithFields(fields Fields) Interface {
	data := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		data[k] = v
	}
	for k, v := range fields {
		data[k] = v
	}
	return &FiledLogger{
		l:      l.l,
		fields: data,
	}
}

// WithError creates an FiledLogger from the logger and adds an error to it, using the value defined in error as key.
func (l *FiledLogger) WithError(err error) Interface {
	if err == nil {
		return l
	}
	return l.WithFields(Fields{"error": err.Error()})
}

// WithFloat64Field creates an FiledLogger from the logger and adds an float64 field to it.
func (l *FiledLogger) WithFloat64Field(key string, value ...float64) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, fmt.Sprintf("%f", v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithFloat32Field creates an FiledLogger from the logger and adds an float32 field to it.
func (l *FiledLogger) WithFloat32Field(key string, value ...float32) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, fmt.Sprintf("%f", v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithInt64Field creates an FiledLogger from the logger and adds an int64 field to it.
func (l *FiledLogger) WithInt64Field(key string, value ...int64) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, strconv.FormatInt(v, 10))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithIntField creates an FiledLogger from the logger and adds an int field to it.
func (l *FiledLogger) WithIntField(key string, value ...int) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, strconv.Itoa(v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithBoolField creates an FiledLogger from the logger and adds an bool field to it.
func (l *FiledLogger) WithBoolField(key string, value ...bool) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, strconv.FormatBool(v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithDurationField creates an FiledLogger from the logger and adds an duration field to it.
func (l *FiledLogger) WithDurationField(key string, value ...time.Duration) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, v.String())
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// SetOutput Set the output of logger
func (l *FiledLogger) SetOutput(out io.Writer) {
	l.l.SetOutput(out)
}

// SetLevel Set the minimum log level that can be displayed.
// Logs below this level will not be displayed.
func (l *FiledLogger) SetLevel(level Level) {
	l.l.SetLevel(level)
}

// SetEncoder set encoder for logging
func (l *FiledLogger) SetEncoder(encoder Encoder) {
	l.l.SetEncoder(encoder)
}

// SetTimeEncoder set time encoder for logger
func (l *FiledLogger) SetTimeEncoder(encoder TimeEncoder) {
	l.l.SetTimeEncoder(encoder)
}

// SetCallerEncoder set caller encoder for logger
func (l *FiledLogger) SetCallerEncoder(encoder CallerEncoder) {
	l.l.SetCallerEncoder(encoder)
}

// Output fake output is just call l.l.Output
func (l *FiledLogger) Output(level Level, calldepth int, fileds Fields, s string) error {
	return l.l.Output(level, calldepth+1, fileds, s)
}

// WithContext Call l.WithFields (fields) to create a logger and inject it into the context
func (l *FiledLogger) WithContext(ctx context.Context, opts ...func(Fields)) context.Context {
	fields := make(Fields)
	for _, opt := range opts {
		opt(fields)
	}
	return context.WithValue(ctx, loggerKey, l.WithFields(fields))
}

// Fields  get fields
func (l *FiledLogger) Fields() Fields {
	f := make(Fields, len(l.fields))
	for k, v := range l.fields {
		f[k] = v
	}
	return f
}

// Flush flush buffer
func (l *FiledLogger) Flush() {
	l.l.Flush()
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Printf(format string, v ...interface{}) {
	_ = l.Output(InfoLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *FiledLogger) Print(v ...interface{}) {
	_ = l.Output(InfoLevel, 2, l.fields, fmt.Sprint(v...))
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *FiledLogger) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(InfoLevel, 2, l.fields, s[:len(s)-1])
}

// Trace calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Trace(v ...interface{}) {
	_ = l.Output(TraceLevel, 2, l.fields, fmt.Sprint(v...))
}

// Tracef calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Tracef(format string, v ...interface{}) {
	_ = l.Output(TraceLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Traceln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Traceln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(TraceLevel, 2, l.fields, s[:len(s)-1])
}

// Debug calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Debug(v ...interface{}) {
	_ = l.Output(DebugLevel, 2, l.fields, fmt.Sprint(v...))
}

// Debugf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Debugf(format string, v ...interface{}) {
	_ = l.Output(DebugLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Debugln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Debugln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(DebugLevel, 2, l.fields, s[:len(s)-1])
}

// Info calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Info(v ...interface{}) {
	_ = l.Output(InfoLevel, 2, l.fields, fmt.Sprint(v...))
}

// Infof calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Infof(format string, v ...interface{}) {
	_ = l.Output(InfoLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Infoln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Infoln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(InfoLevel, 2, l.fields, s[:len(s)-1])
}

// Warn calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warn(v ...interface{}) {
	_ = l.Output(WarnLevel, 2, l.fields, fmt.Sprint(v...))
}

// Warnf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warnf(format string, v ...interface{}) {
	_ = l.Output(WarnLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Warnln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warnln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(WarnLevel, 2, l.fields, s[:len(s)-1])
}

// Warning calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warning(v ...interface{}) {
	_ = l.Output(WarnLevel, 2, l.fields, fmt.Sprint(v...))
}

// Warningf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warningf(format string, v ...interface{}) {
	_ = l.Output(WarnLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Warningln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Warningln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(WarnLevel, 2, l.fields, s[:len(s)-1])
}

// Error calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Error(v ...interface{}) {
	_ = l.Output(ErrorLevel, 2, l.fields, fmt.Sprint(v...))
}

// Errorf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Errorf(format string, v ...interface{}) {
	_ = l.Output(ErrorLevel, 2, l.fields, fmt.Sprintf(format, v...))
}

// Errorln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *FiledLogger) Errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(ErrorLevel, 2, l.fields, s[:len(s)-1])
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *FiledLogger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = l.Output(PanicLevel, 2, l.fields, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *FiledLogger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = l.Output(PanicLevel, 2, l.fields, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *FiledLogger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(PanicLevel, 2, l.fields, s[:len(s)-1])
	panic(s)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *FiledLogger) Fatal(v ...interface{}) {
	_ = l.Output(FatalLevel, 2, l.fields, fmt.Sprint(v...))
	l.Flush()
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *FiledLogger) Fatalf(format string, v ...interface{}) {
	_ = l.Output(FatalLevel, 2, l.fields, fmt.Sprintf(format, v...))
	l.Flush()
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *FiledLogger) Fatalln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(FatalLevel, 2, l.fields, s[:len(s)-1])
	l.Flush()
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
	return func(l *config) {
		l.level = level
	}
}

// AddEncoder  wrap the encoder as an option
func AddEncoder(enc Encoder) Option {
	return func(l *config) {
		l.encoder = enc
	}
}

// AddTimeEncoder wrap the time encoder as an option
func AddTimeEncoder(enc TimeEncoder) Option {
	return func(l *config) {
		l.timeEncoder = enc
	}
}

// AddCallerEncoder wrap the caller encoder as an option
func AddCallerEncoder(enc CallerEncoder) Option {
	return func(l *config) {
		l.callerEncoder = enc
	}
}

// AddBufferSize wrap the caller encoder as an option
func AddBufferSize(size int) Option {
	return func(l *config) {
		l.bufferSize = size
	}
}

// AddFlushInterval wrap the caller encoder as an option
func AddFlushInterval(interval time.Duration) Option {
	return func(l *config) {
		l.flushInterval = interval
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
	return std
}

// WithContext Inject logger into the context
func WithContext(ctx context.Context, opts ...func(Fields)) context.Context {
	return std.WithContext(ctx, opts...)
}

// FromContext get logger forom context
func FromContext(ctx context.Context) Interface {
	if ctx == nil {
		panic("nil context")
	}
	logger, ok := ctx.Value(loggerKey).(Interface)
	if !ok {
		logger = std
	}
	return logger
}

// New create Interface and set output is w
func New(w io.Writer, opts ...Option) *Logger {
	c := &config{
		level:         InfoLevel,
		bufferSize:    defaultBufSize,
		encoder:       DefaultTextEncoder,
		flushInterval: defaultFlushInterval,
	}
	// apply option
	for _, opt := range opts {
		opt(c)
	}

	l := &Logger{
		config:      c,
		stopFlush:   make(chan struct{}),
		bufioWriter: bufio.NewWriterSize(w, c.bufferSize),
	}

	// Close the ticker after gc and exit the coroutine
	runtime.SetFinalizer(l, func(l *Logger) { close(l.stopFlush) })

	go l.launchFlushDaemon()

	return l
}

func (l *Logger) launchFlushDaemon() {
	ticker := time.NewTicker(l.config.flushInterval)
	for {
		select {
		case <-ticker.C:
			l.Flush()
		case <-l.stopFlush:
			l.Flush()
			ticker.Stop()
			return
		}
	}
}

// WithContext Inject logger into context
func (l *Logger) WithContext(ctx context.Context, opts ...func(Fields)) context.Context {
	fields := make(Fields)
	for _, opt := range opts {
		opt(fields)
	}
	return context.WithValue(ctx, loggerKey, l.WithFields(fields))
}

// SetLevel Set the minimum log level that can be displayed.
// Logs below this level will not be displayed.
func (l *Logger) SetLevel(level Level) {
	l.locker.Lock()
	defer l.locker.Unlock()
	l.config.level = level
}

// SetEncoder set encoder for logging
func (l *Logger) SetEncoder(encoder Encoder) {
	l.locker.Lock()
	defer l.locker.Unlock()
	l.config.encoder = encoder
}

// SetTimeEncoder set time encoder for loger
func (l *Logger) SetTimeEncoder(encoder TimeEncoder) {
	l.locker.Lock()
	defer l.locker.Unlock()
	l.config.timeEncoder = encoder
}

// SetCallerEncoder set caller encoder for logger
func (l *Logger) SetCallerEncoder(encoder CallerEncoder) {
	l.locker.Lock()
	defer l.locker.Unlock()
	l.config.callerEncoder = encoder
}

// Output call the format function and write the formatted data to io.Writer
func (l *Logger) Output(level Level, calldepth int, fileds Fields, s string) error {
	var (
		caller   string
		dateTime string
	)
	l.locker.Lock()
	if l.level > level { // skip output by level limit
		l.locker.Unlock()
		return nil
	}
	callerEncoder := l.config.callerEncoder // Just copy the value of the pointer
	timeEncoder := l.config.timeEncoder     // Just copy the value of the pointer
	l.locker.Unlock()
	if timeEncoder != nil {
		t := time.Now() // Sacrificing time accuracy, but gaining performance
		dateTime = timeEncoder(&t)
	}
	if callerEncoder != nil {
		_, file, line, ok := runtime.Caller(calldepth) // Expensive time consumption
		if !ok {
			file = "???"
			line = 0
		}
		caller = callerEncoder(file, line)
	}
	buf := bufPool.Get()
	defer bufPool.Put(buf)
	if err := l.config.encoder.Encode(buf, &Message{
		Level:   level,
		Time:    dateTime,
		Caller:  caller,
		Fields:  fileds,
		Message: s,
	}); err != nil {
		return err
	}
	l.locker.Lock()
	defer l.locker.Unlock()
	_, err := l.bufioWriter.Write(*buf)
	return err
}

// Fields  get fields
func (l *Logger) Fields() Fields { return nil }

// Flush flush the buffer to the file
func (l *Logger) Flush() {
	l.locker.Lock()
	defer l.locker.Unlock()
	if err := l.bufioWriter.Flush(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error flushing data to writer %s", err)
	}
}

// SetOutput set the Logger writer to out io.Writer
func (l *Logger) SetOutput(out io.Writer) {
	l.locker.Lock()
	defer l.locker.Unlock()
	if err := l.bufioWriter.Flush(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error flushing data to writer %s", err)
	}
	l.bufioWriter = bufio.NewWriterSize(out, l.bufferSize)
}

// WithField key is string value is string,
// this key value will be printed out every time when log is printed
func (l *Logger) WithField(key string, value ...string) Interface {
	return l.WithFields(Fields{key: strings.Join(value, ",")})
}

// WithError key is string value is Error,
// this key value will be printed out every time when log is printed
func (l *Logger) WithError(err error) Interface {
	return l.WithFields(Fields{"error": err.Error()})
}

// WithFloat64Field key is string value is float64,
// this key value will be printed out every time when log is printed
func (l *Logger) WithFloat64Field(key string, value ...float64) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, fmt.Sprintf("%f", v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithFloat32Field key is string value is float32,
// this key value will be printed out every time when log is printed
func (l *Logger) WithFloat32Field(key string, value ...float32) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, fmt.Sprintf("%f", v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithInt64Field key is string value is int64,
// this key value will be printed out every time when log is printed
func (l *Logger) WithInt64Field(key string, value ...int64) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, strconv.FormatInt(v, 10))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithIntField key is string value is int,
// this key value will be printed out every time when log is printed
func (l *Logger) WithIntField(key string, value ...int) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, strconv.Itoa(v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithDurationField key is string value is time.Duration,
// this key value will be printed out every time when log is printed
func (l *Logger) WithDurationField(key string, value ...time.Duration) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, v.String())
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithBoolField key is string value is bool,
// this key value will be printed out every time when log is printed
func (l *Logger) WithBoolField(key string, value ...bool) Interface {
	values := make([]string, 0, len(value))
	for _, v := range value {
		values = append(values, strconv.FormatBool(v))
	}
	return l.WithFields(Fields{key: strings.Join(values, ",")})
}

// WithFields Parameter is map[string] string type
// this key value will be printed out every time when log is printed
func (l *Logger) WithFields(fields Fields) Interface {
	data := make(Fields, len(fields))
	for k, v := range fields {
		data[k] = v
	}
	return &FiledLogger{
		l:      l,
		fields: data,
	}
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	_ = l.Output(InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) {
	_ = l.Output(InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(InfoLevel, 2, nil, s[:len(s)-1])
}

// Trace calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Trace(v ...interface{}) {
	_ = l.Output(TraceLevel, 2, nil, fmt.Sprint(v...))
}

// Tracef calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Tracef(format string, v ...interface{}) {
	_ = l.Output(TraceLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Traceln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Traceln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(TraceLevel, 2, nil, s[:len(s)-1])
}

// Debug calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debug(v ...interface{}) {
	_ = l.Output(DebugLevel, 2, nil, fmt.Sprint(v...))
}

// Debugf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debugf(format string, v ...interface{}) {
	_ = l.Output(DebugLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Debugln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debugln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(DebugLevel, 2, nil, s[:len(s)-1])
}

// Info calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Info(v ...interface{}) {
	_ = l.Output(InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Infof calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infof(format string, v ...interface{}) {
	_ = l.Output(InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Infoln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infoln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(InfoLevel, 2, nil, s[:len(s)-1])
}

// Warn calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warn(v ...interface{}) {
	_ = l.Output(WarnLevel, 2, nil, fmt.Sprint(v...))
}

// Warnf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warnf(format string, v ...interface{}) {
	_ = l.Output(WarnLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Warnln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warnln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(WarnLevel, 2, nil, s[:len(s)-1])
}

// Warning calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warning(v ...interface{}) {
	_ = l.Output(WarnLevel, 2, nil, fmt.Sprint(v...))
}

// Warningf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningf(format string, v ...interface{}) {
	_ = l.Output(WarnLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Warningln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(WarnLevel, 2, nil, s[:len(s)-1])
}

// Error calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Error(v ...interface{}) {
	_ = l.Output(ErrorLevel, 2, nil, fmt.Sprint(v...))
}

// Errorf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorf(format string, v ...interface{}) {
	_ = l.Output(ErrorLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Errorln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(ErrorLevel, 2, nil, s[:len(s)-1])
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = l.Output(PanicLevel, 2, nil, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = l.Output(PanicLevel, 2, nil, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(PanicLevel, 2, nil, s[:len(s)-1])
	panic(s)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	_ = l.Output(FatalLevel, 2, nil, fmt.Sprint(v...))
	l.Flush()
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	_ = l.Output(FatalLevel, 2, nil, fmt.Sprintf(format, v...))
	l.Flush()
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.Output(FatalLevel, 2, nil, s[:len(s)-1])
	l.Flush()
	os.Exit(1)
}

// SetOutput set output to out
func SetOutput(out io.Writer) {
	std.SetOutput(out)
}

// SetEncoder set Encoder to e for std logger
func SetEncoder(e Encoder) {
	std.SetEncoder(e)
}

// SetTimeEncoder set time encoder for std logger
func SetTimeEncoder(e TimeEncoder) {
	std.SetTimeEncoder(e)
}

// SetCallerEncoder set caller encoder for std logger
func SetCallerEncoder(e CallerEncoder) {
	std.SetCallerEncoder(e)
}

// WithFloat64Field key is string value is float64,
// this key value will be printed out every time when log is printed
func WithFloat64Field(key string, value float64) Interface {
	return std.WithFloat64Field(key, value)
}

// WithFloat32Field key is string value is float32,
// this key value will be printed out every time when log is printed
func WithFloat32Field(key string, value float32) Interface {
	return std.WithFloat32Field(key, value)
}

// WithInt64Field key is string value is int64,
// this key value will be printed out every time when log is printed
func WithInt64Field(key string, value int64) Interface {
	return std.WithInt64Field(key, value)
}

// WithIntField key is string value is int,
// this key value will be printed out every time when log is printed
func WithIntField(key string, value int) Interface {
	return std.WithIntField(key, value)
}

// WithDurationField key is string value is int,
// this key value will be printed out every time when log is printed
func WithDurationField(key string, value time.Duration) Interface {
	return std.WithDurationField(key, value)
}

// WithBoolField key is string value is bool,
// this key value will be printed out every time when log is printed
func WithBoolField(key string, value bool) Interface {
	return std.WithBoolField(key, value)
}

// WithField key is string value is string,
// this key value will be printed out every time when log is printed
func WithField(key string, value ...string) Interface {
	return std.WithField(key, value...)
}

// WithFields Parameter is map[string] string type
// this key value will be printed out every time when log is printed
func WithFields(fields Fields) Interface {
	return std.WithFields(fields)
}

// WithError key is string value is Error,
// this key value will be printed out every time when log is printed
func WithError(err error) Interface {
	return std.WithError(err)
}

// SetLevel Set the log level, logs below this level will not be printed
func SetLevel(level Level) {
	std.SetLevel(level)
}

// Flush flush the buffer data to the file
func Flush() { std.Flush() }

// Printf calls std.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	_ = std.Output(InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Print calls std.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	_ = std.Output(InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Println calls std.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(InfoLevel, 2, nil, s[:len(s)-1])
}

// Trace calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Trace(v ...interface{}) {
	_ = std.Output(TraceLevel, 2, nil, fmt.Sprint(v...))
}

// Tracef calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Tracef(format string, v ...interface{}) {
	_ = std.Output(TraceLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Traceln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Traceln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(TraceLevel, 2, nil, s[:len(s)-1])
}

// Debug calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Debug(v ...interface{}) {
	_ = std.Output(DebugLevel, 2, nil, fmt.Sprint(v...))
}

// Debugf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, v ...interface{}) {
	_ = std.Output(DebugLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Debugln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Debugln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(DebugLevel, 2, nil, s[:len(s)-1])
}

// Info calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Info(v ...interface{}) {
	_ = std.Output(InfoLevel, 2, nil, fmt.Sprint(v...))
}

// Infof calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...interface{}) {
	_ = std.Output(InfoLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Infoln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Infoln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(InfoLevel, 2, nil, s[:len(s)-1])
}

// Warn calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Warn(v ...interface{}) {
	_ = std.Output(WarnLevel, 2, nil, fmt.Sprint(v...))
}

// Warnf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Warnf(format string, v ...interface{}) {
	_ = std.Output(WarnLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Warnln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Warnln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(WarnLevel, 2, nil, s[:len(s)-1])
}

// Error calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Error(v ...interface{}) {
	_ = std.Output(ErrorLevel, 2, nil, fmt.Sprint(v...))
}

// Errorf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...interface{}) {
	_ = std.Output(ErrorLevel, 2, nil, fmt.Sprintf(format, v...))
}

// Errorln calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(ErrorLevel, 2, nil, s[:len(s)-1])
}

// Panic is equivalent to l.Print() followed by a call to panic().
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = std.Output(PanicLevel, 2, nil, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = std.Output(PanicLevel, 2, nil, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(PanicLevel, 2, nil, s[:len(s)-1])
	panic(s)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	_ = std.Output(FatalLevel, 2, nil, fmt.Sprint(v...))
	std.Flush()
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	_ = std.Output(FatalLevel, 2, nil, fmt.Sprintf(format, v...))
	std.Flush()
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func Fatalln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = std.Output(FatalLevel, 2, nil, s[:len(s)-1])
	std.Flush()
	os.Exit(1)
}
