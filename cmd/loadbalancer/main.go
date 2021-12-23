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

package main

import (
	"kunstack.com/pharos/pkg/safe"
	"math/rand"
	"time"

	"kunstack.com/pharos/cmd/loadbalancer/app"
	"kunstack.com/pharos/pkg/log"
)

func main() {
	l := log.WithoutContext()
	defer l.Flush()
	rand.Seed(time.Now().UnixNano())
	stopChan := safe.SetupSignalHandler()
	cmd := app.NewLoadBalancerCommand(stopChan)
	if err := cmd.Execute(); err != nil {
		l.Fatalln(err)
	}
}
