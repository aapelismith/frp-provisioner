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

package safe

import (
	"context"
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"io"
	"sync"
)

// TaskManager this Interface is designed to manage single-run tasks in the system
// the purpose of the design is to facilitate the unified management of coroutine and
// avoid its leakage all coroutine should be exited after CTX is cleared by cancel
type TaskManager interface {
	io.Closer
	// Ctx get base context from TaskManger
	Ctx() context.Context
	// GoCtx start goroutine with context
	GoCtx(goroutine func(ctx context.Context))
}

// taskManager implement TaskManager
type taskManager struct {
	lock      sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup
}

// Ctx get base context from TaskManger
func (p *taskManager) Ctx() context.Context {
	return p.ctx
}

// GoCtx start goroutine with context
func (p *taskManager) GoCtx(goroutine func(ctx context.Context)) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.waitGroup.Add(1)
	Go(func() {
		defer p.waitGroup.Done()
		goroutine(p.ctx)
	})
}

// Close tsk manager
func (p *taskManager) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.cancel()
	p.waitGroup.Wait()
	return nil
}

// NewTaskManager create new task manager with context
func NewTaskManager(parentCtx context.Context) TaskManager {
	ctx, cancel := context.WithCancel(parentCtx)
	return &taskManager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Go create goroutine with default recover function
func Go(goroutine func()) {
	GoWithRecovery(goroutine, log.WithoutContext().Sugar().Panic)
}

// GoWithRecovery create goroutine with custom recover function
func GoWithRecovery(goroutine func(), customRecover func(args ...any)) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				customRecover(err)
			}
		}()
		goroutine()
	}()
}
