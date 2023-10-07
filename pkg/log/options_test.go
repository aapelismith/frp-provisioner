/*
 * Copyright 2021 Aapeli.Smith<aapeli.nian@gmail.com>.
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
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"github.com/spf13/pflag"
	"testing"
)

func TestOptions_AddFlags(t *testing.T) {
	args := []string{
		"--log.level=info",
		"--log.development",
		"--log.disable-caller",
		"--log.disable-stacktrace",
		"--log.encoding=json",
	}

	options := log.NewOptions()
	options.SetDefaults()

	cleanFlags := pflag.NewFlagSet("", pflag.ContinueOnError)
	options.AddFlags(cleanFlags)

	if err := cleanFlags.Parse(args); err != nil {
		t.Fatal(err)
	}

	if err := cleanFlags.Parse(args); err != nil {
		t.Fatal(err)
	}

	if options.Level.String() != "info" {
		t.Fatalf("expected 'info'; got %v", options.Level.String())
	}

	if options.Development != true {
		t.Fatalf("expected 'true'; got %v", options.Development)
	}

	if options.DisableCaller != true {
		t.Fatalf("expected 'true'; got %v", options.DisableCaller)
	}

	if options.DisableStacktrace != true {
		t.Fatalf("expected 'true'; got %v", options.DisableStacktrace)
	}

	if options.Encoding != "json" {
		t.Fatalf("expected 'json'; got %v", options.Encoding)
	}
}
