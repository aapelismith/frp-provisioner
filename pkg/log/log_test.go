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

package log_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"kunstack.com/pharos/pkg/log"
)

// These tests are too simple.

const (
	CallerRegex = `file=".*/[A-Za-z0-9_\-]+\.go:57"`
)

func TestOutput(t *testing.T) {
	const testString = "test"
	defer log.Flush()
	var b bytes.Buffer
	l := log.New(
		os.Stderr, log.AddLevel(log.TraceLevel),
		log.AddTextEncoder(),
		log.AddFullCaller(),
	)
	defer l.Flush()
	l.SetOutput(&b)
	l.Println(testString)
	expect := fmt.Sprintf(`msg="%s"`, testString) + "\n"
	l.Flush()
	if !strings.HasSuffix(b.String(), expect) {
		t.Errorf("log output should match %s is %s", expect, b.String())
	}
	l.WithField("hello", "world").Println(testString)
}

func TestWithCaller(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, log.AddFullCaller())
	l.Println("test")
	l.Flush()
	ok, err := regexp.MatchString(CallerRegex, buf.String())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("caller does not match expectations")
	}
}

func TestParseLevel(t *testing.T) {
	tester := []struct {
		Value string
		Want  log.Level
	}{
		{Value: "INFO", Want: log.InfoLevel},
		{Value: "Info", Want: log.InfoLevel},
		{Value: "INfo", Want: log.InfoLevel},
		{Value: "InfO", Want: log.InfoLevel},
		{Value: "trace", Want: log.TraceLevel},
		{Value: "debug", Want: log.DebugLevel},
		{Value: "warn", Want: log.WarnLevel},
		{Value: "error", Want: log.ErrorLevel},
		{Value: "panic", Want: log.PanicLevel},
		{Value: "fatal", Want: log.FatalLevel},
		{Value: "off", Want: log.OffLevel},
	}
	for _, test := range tester {
		lvl, err := log.ParseLevel(test.Value)
		if err != nil {
			t.Fatal(err)
		}
		if lvl != test.Want {
			t.Fatalf("level is not consistent with expectations, want: %v, current: %v", test.Want, lvl)
		}
	}
	lvl, err := log.ParseLevel("not found level")
	if err == nil {
		t.Fatalf("err is not consistent with expectations, want: err != nil, current: %v", err)
	}
	if lvl != 0 {
		t.Fatalf("err is not consistent with expectations, want: \"\", current: %v", lvl)
	}
}

func TestEmptyPrintCreatesLine(t *testing.T) {
	var b bytes.Buffer
	l := log.New(&b)
	l.Print()
	l.Println("non-empty")
	l.Flush()
	output := b.String()
	if n := strings.Count(output, "\n"); n != 2 {
		t.Errorf("expected 2 lines, got %d", n)
	}
}

func TestJSONEncoder(t *testing.T) {
	var buf bytes.Buffer
	var testString = "test"
	var testMap = map[string]string{}
	l := log.New(&buf, log.AddJSONEncoder())
	l.SetEncoder(log.DefaultJSONEncoder)
	l.Println(testString)
	l.Flush()
	t.Logf("%s", buf.String())
	if err := json.Unmarshal(buf.Bytes(), &testMap); err != nil {
		t.Fatal(err)
	}
	val, ok := testMap[log.DefaultJSONEncoder.KeyMessage]
	if !ok {
		t.Fatal("testMap[log.KeyMsg] is not consistent with expectations")
	}
	if val != testString {
		t.Fatalf("val is not consistent with expectations, want: %v, current: %v", testString, val)
	}
}

func BenchmarkPrintln(b *testing.B) {
	const testString = "test"
	var buf bytes.Buffer
	l := log.New(&buf)
	defer l.Flush()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Println(testString)
	}
}

func BenchmarkPrintAddTime(b *testing.B) {
	const testString = "test"
	var buf bytes.Buffer
	l := log.New(&buf, log.AddTimeRFC3339())
	defer l.Flush()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Print(testString)
	}
}

func BenchmarkPrintWithCaller(b *testing.B) {
	const testString = "test"
	var buf bytes.Buffer
	l := log.New(&buf, log.AddShortCaller())
	defer l.Flush()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Print(testString)
	}
}
