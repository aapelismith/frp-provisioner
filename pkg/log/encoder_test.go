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
	"encoding/json"
	"testing"
	"time"

	"kunstack.com/pharos/pkg/log"
)

func TestEncoder(t *testing.T) {
	t.Run("test json encoder", func(t *testing.T) {
		var buf []byte
		wantTime := time.Now().String()
		wantCaller := "/path/example/main.go:123"
		wantField := log.Fields{"hello": "this is want string", "time": "this is duplicate time field"}
		wantMsg := "this is want msg"

		e := &log.JSONEncoder{
			KeyTime:            "time",
			KeyLevel:           "level",
			KeyCaller:          "file",
			KeyMessage:         "msg",
			DuplicateKeyPrefix: "fields.",
		}

		msg := &log.Message{
			Level:   log.DebugLevel,
			Time:    wantTime,
			Caller:  wantCaller,
			Fields:  wantField,
			Message: wantMsg,
		}

		if err := e.Encode(&buf, msg); err != nil {
			t.Fatal(err)
		}

		targetMap := map[string]interface{}{}

		if err := json.Unmarshal(buf, &targetMap); err != nil {
			t.Fatal(err)
		}

		tm, ok := targetMap[e.KeyTime]
		if !ok {
			t.Fatalf("want true, got false")
		}

		if tm != wantTime {
			t.Fatalf("want: %s got: %s", tm, wantTime)
		}

		lvl, ok := targetMap[e.KeyLevel]
		if !ok {
			t.Fatalf("want true, got false")
		}

		if lvl != "DEBUG" {
			t.Fatalf("want: DEBUG got: %s", lvl)
		}

		m, ok := targetMap[e.KeyMessage]
		if !ok {
			t.Fatalf("want true, got false")
		}

		if m != wantMsg {
			t.Fatalf("want: %s got: %s", m, wantMsg)
		}

		file, ok := targetMap[e.KeyCaller]
		if !ok {
			t.Fatalf("want true, got false")
		}

		if file != wantCaller {
			t.Fatalf("want: %s got: %s", file, wantCaller)
		}

		for k, v := range wantField {
			value1, ok1 := targetMap[k]
			value2, ok2 := targetMap[e.DuplicateKeyPrefix+k]
			if !ok1 && !ok2 {
				t.Fatalf("want true, got false")
			}
			if value1 != v && value2 != v {
				t.Fatalf("want: value1=%s or value2=%s, but got: value1=%s or value2=%s", value1, value2, value1, value2)
			}
		}
		t.Logf("log output is %s", buf)
	})

	t.Run("test text encoder", func(t *testing.T) {
		var buf []byte
		wantTime := time.Now().String()
		wantCaller := "/path/example/main.go:123"
		wantField := log.Fields{"hello": "this is want string", "time": "this is duplicate time field"}
		wantMsg := "this is want msg"

		e := &log.TextEncoder{
			KeyTime:            "time",
			KeyLevel:           "level",
			KeyCaller:          "file",
			KeyMessage:         "msg",
			DuplicateKeyPrefix: "fields.",
		}

		msg := &log.Message{
			Level:   log.DebugLevel,
			Time:    wantTime,
			Caller:  wantCaller,
			Fields:  wantField,
			Message: wantMsg,
		}

		if err := e.Encode(&buf, msg); err != nil {
			t.Fatal(err)
		}

		// TODO: do some check

		t.Logf("log output is %s", buf)
	})
}
