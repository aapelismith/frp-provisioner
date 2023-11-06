/*
 * Copyright 2021 The Frp Sig Authors.
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

package main

import (
	"github.com/frp-sigs/frp-provisioner/cmd/manager/app"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	l := log.WithoutContext().Sugar()
	stopCtx := signals.SetupSignalHandler()
	cmd := app.NewManagerCommand(stopCtx)

	if err := cmd.Execute(); err != nil {
		l.Fatalln(err)
	}
}
