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
	"context"
	"github.com/aapelismith/frp-service-provider/pkg/log"
	"testing"
)

func Test_NewLogger(t *testing.T) {
	opts := log.NewOptions()
	opts.SetDefaults()
	l, err := log.NewLogger(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	l.Sugar().Debugf("hello %s", "world")
}

func Test_LogContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := log.NewOptions()
	opts.SetDefaults()

	l, err := log.NewLogger(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx = log.NewContext(ctx, l)

	log.FromContext(ctx).Sugar().Info("hello world")
}
