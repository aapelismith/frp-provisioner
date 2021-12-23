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

import "sync"

const maxBufSize = 1 << 19

type bytesBuffer struct {
	pool *sync.Pool
}

// Get gets an empty buf pointer from the buffer
func (b *bytesBuffer) Get() *[]byte {
	return b.pool.Get().(*[]byte)
}

// Put puts the buf array back to buffer
func (b *bytesBuffer) Put(buf *[]byte) {
	if cap(*buf) > maxBufSize {
		return
	}
	tmp := (*buf)[:0]
	b.pool.Put(&tmp)
}

// NewBytesBuffer creates a new buffer and returns a pointer by default
// Point to an empty underlying array with a length of 2024 and a slice with length 0
func NewBytesBuffer() *bytesBuffer {
	return &bytesBuffer{
		pool: &sync.Pool{
			New: func() interface{} {
				s := make([]byte, 0, 1<<11)
				return &s
			},
		},
	}
}
