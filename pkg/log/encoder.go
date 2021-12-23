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
	"strconv"
)

var (
	bufPool         = NewBytesBuffer()
	_       Encoder = &TextEncoder{}
	_       Encoder = &JSONEncoder{}
	// DefaultTextEncoder the default log encoder
	DefaultTextEncoder = &TextEncoder{
		KeyTime:            "time",
		KeyLevel:           "level",
		KeyCaller:          "file",
		KeyMessage:         "msg",
		DuplicateKeyPrefix: "fields.",
	}
	// DefaultJSONEncoder the default json encoder
	DefaultJSONEncoder = &JSONEncoder{
		KeyTime:            "time",
		KeyLevel:           "level",
		KeyCaller:          "file",
		KeyMessage:         "msg",
		DuplicateKeyPrefix: "fields.",
	}
)

// Message log message structure
type Message struct {
	Level   Level
	Time    string
	Caller  string
	Fields  Fields
	Message string
}

// Encoder Format Encoder
type Encoder interface {
	Encode(w *[]byte, msg *Message) error
}

// TextEncoder Format logs to text
type TextEncoder struct {
	KeyTime            string
	KeyLevel           string
	KeyCaller          string
	KeyMessage         string
	DuplicateKeyPrefix string
}

// Encode format logs to text
func (e *TextEncoder) Encode(buf *[]byte, msg *Message) error {
	if msg.Time != "" {
		*buf = append(*buf, e.KeyTime+"="+strconv.Quote(msg.Time)...)
		*buf = append(*buf, ' ')
	}
	*buf = append(*buf, e.KeyLevel+"="...)
	switch msg.Level {
	case TraceLevel:
		*buf = append(*buf, `"TRACE"`...)
	case DebugLevel:
		*buf = append(*buf, `"DEBUG"`...)
	case InfoLevel:
		*buf = append(*buf, `"INFO"`...)
	case WarnLevel:
		*buf = append(*buf, `"WARN"`...)
	case ErrorLevel:
		*buf = append(*buf, `"ERROR"`...)
	case PanicLevel:
		*buf = append(*buf, `"PANIC"`...)
	case FatalLevel:
		*buf = append(*buf, `"FATAL"`...)
	default:
		*buf = append(*buf, `"UNKNOWN"`...)
	}
	*buf = append(*buf, ' ')

	if msg.Caller != "" {
		*buf = append(*buf, e.KeyCaller+"="+strconv.Quote(msg.Caller)...)
		*buf = append(*buf, ' ')
	}
	for key, value := range msg.Fields {
		switch key {
		case e.KeyMessage, e.KeyCaller, e.KeyLevel, e.KeyTime:
			key = e.DuplicateKeyPrefix + key
		}
		*buf = append(*buf, key+"="+strconv.Quote(value)...)
		*buf = append(*buf, ' ')
	}
	if msg.Message != "" {
		*buf = append(*buf, e.KeyMessage+"="+strconv.Quote(msg.Message)...)
	} else {
		*buf = (*buf)[:len(*buf)-1]
	}
	*buf = append(*buf, '\n')
	return nil
}

// JSONEncoder Format log to json
type JSONEncoder struct {
	KeyTime            string
	KeyLevel           string
	KeyCaller          string
	KeyMessage         string
	DuplicateKeyPrefix string
}

// Encode Format log to json
func (e *JSONEncoder) Encode(buf *[]byte, msg *Message) error {
	data := make(Fields, 4+len(msg.Fields))
	if msg.Time != "" {
		data[e.KeyTime] = msg.Time
	}
	switch msg.Level {
	case TraceLevel:
		data[e.KeyLevel] = "TRACE"
	case DebugLevel:
		data[e.KeyLevel] = "DEBUG"
	case InfoLevel:
		data[e.KeyLevel] = "INFO"
	case WarnLevel:
		data[e.KeyLevel] = "WARN"
	case ErrorLevel:
		data[e.KeyLevel] = "ERROR"
	case PanicLevel:
		data[e.KeyLevel] = "PANIC"
	case FatalLevel:
		data[e.KeyLevel] = "FATAL"
	default:
		data[e.KeyLevel] = "UNKNOWN"
	}
	if msg.Caller != "" {
		data[e.KeyCaller] = msg.Caller
	}
	for key, value := range msg.Fields {
		switch key {
		case e.KeyMessage, e.KeyCaller, e.KeyLevel, e.KeyTime:
			key = e.DuplicateKeyPrefix + key
		}
		data[key] = value
	}
	if msg.Message != "" {
		data[e.KeyMessage] = msg.Message
	}
	*buf = append(*buf, '{')
	for key, value := range data {
		*buf = append(*buf, strconv.Quote(key)+":"+strconv.Quote(value)...)
		*buf = append(*buf, ',')
	}
	*buf = (*buf)[:len(*buf)-1]
	*buf = append(*buf, "}\n"...)
	return nil
}
