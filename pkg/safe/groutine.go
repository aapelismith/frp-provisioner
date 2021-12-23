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

package safe

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"

	"kunstack.com/pharos/pkg/log"
)

type routine struct {
	groutine  func(stopChain <-chan struct{})
	stopChain chan struct{}
}

// ServiceManager struct for manage groutine
// This structure is designed to manage the long-running services of the system
type ServiceManager struct {
	routines   []routine
	routineCtx []func(ctx context.Context)
	waitGroup  sync.WaitGroup
	lock       sync.Mutex
	baseCtx    context.Context
	baseCancel context.CancelFunc
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
}

// NewServiceManager create ServiceManager
func NewServiceManager(parentCtx context.Context) *ServiceManager {
	baseCtx, baseCancel := context.WithCancel(parentCtx)
	ctx, cancel := context.WithCancel(baseCtx)
	return &ServiceManager{
		baseCtx:    baseCtx,
		baseCancel: baseCancel,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Ctx get ServiceManager base context
func (p *ServiceManager) Ctx() context.Context {
	return p.baseCtx
}

// Start the ServiceManager
func (p *ServiceManager) Start() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.baseCtx.Err() != nil {
		return p.baseCtx.Err()
	}
	if p.started {
		return errors.New("the ServiceManager has been started")
	}
	p.ctx, p.cancel = context.WithCancel(p.baseCtx)
	for _, routine := range p.routines {
		p.waitGroup.Add(1)
		routine.stopChain = make(chan struct{})
		Go(func() {
			defer p.waitGroup.Done()
			routine.groutine(routine.stopChain)
		})
	}
	for _, routine := range p.routineCtx {
		p.waitGroup.Add(1)
		Go(func() {
			defer p.waitGroup.Done()
			routine(p.ctx)
		})
	}
	return nil
}

// Stop all service
func (p *ServiceManager) Stop() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.cancel()
	for _, routine := range p.routines {
		routine.stopChain <- struct{}{}
	}
	p.waitGroup.Wait()
	for _, routine := range p.routines {
		close(routine.stopChain)
	}
	p.started = false
}

// Close ServiceManager, stop all service
func (p *ServiceManager) Close() {
	p.Stop()
	p.baseCancel()
}

// Go create groutine with stopChan
func (p *ServiceManager) Go(groutine func(stopChain <-chan struct{})) {
	stopChain := make(chan struct{})
	p.lock.Lock()
	defer p.lock.Unlock()
	p.routines = append(p.routines, routine{
		groutine:  groutine,
		stopChain: stopChain,
	})
	if p.started {
		p.waitGroup.Add(1)
		Go(func() {
			defer p.waitGroup.Done()
			groutine(stopChain)
		})
	}
}

// GoCtx  create groutine with ctx
func (p *ServiceManager) GoCtx(groutine func(ctx context.Context)) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.routineCtx = append(p.routineCtx, groutine)
	if p.started {
		p.waitGroup.Add(1)
		Go(func() {
			defer p.waitGroup.Done()
			groutine(p.ctx)
		})
	}
}

// Go width recovery
func Go(goroutine func()) {
	GoWithRecovery(
		goroutine,
		func(err interface{}) {
			log.Errorln(err)
			log.Errorf("%s", debug.Stack())
		})
}

// GoWithRecovery .
func GoWithRecovery(goroutine func(), customRecover func(err interface{})) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				customRecover(err)
			}
		}()
		goroutine()
	}()
}

// TaskManager this structure is designed to manage single-run tasks in the system
// the purpose of the design is to facilitate the unified management of coroutine and avoid its leakage
// all coroutine should be exited after CTX is cleared by calcel
type TaskManager struct {
	lock      sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup
}

// Ctx get base context from TaskManegr
func (p *TaskManager) Ctx() context.Context {
	return p.ctx
}

// GoCtx start groutine with context
func (p *TaskManager) GoCtx(groutine func(ctx context.Context)) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.waitGroup.Add(1)
	Go(func() {
		defer p.waitGroup.Done()
		groutine(p.ctx)
	})
}

// Close tsk manager
func (p *TaskManager) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.cancel()
	p.waitGroup.Wait()
}

// NewTaskManager create task manager
func NewTaskManager(parentCtx context.Context) *TaskManager {
	ctx, cancel := context.WithCancel(parentCtx)
	return &TaskManager{
		ctx:    ctx,
		cancel: cancel,
	}
}
