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

package log

import (
	"context"
	"go.uber.org/zap"
)

type loggerKey struct{}

// NewContext create new context with logger
func NewContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext get a zap logger instance from the context
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	l, ok := ctx.Value(loggerKey{}).(*zap.Logger)
	if ok {
		return l
	}
	return zap.L()
}

// WithoutContext get a zap logger instance without context
func WithoutContext() *zap.Logger {
	return FromContext(context.Background())
}

// ReplaceGlobals replace the global logger with l
func ReplaceGlobals(l *zap.Logger) {
	zap.ReplaceGlobals(l)
}

// NewLogger create zap logger via Options
func NewLogger(ctx context.Context, opt *Options) (*zap.Logger, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config := zap.Config{
		Level:             opt.Level,
		Development:       opt.Development,
		DisableCaller:     opt.DisableCaller,
		DisableStacktrace: opt.DisableStacktrace,
		Sampling:          opt.Sampling,
		Encoding:          opt.Encoding,
		EncoderConfig:     opt.EncoderConfig,
		OutputPaths:       opt.OutputPaths,
		ErrorOutputPaths:  opt.ErrorOutputPaths,
		InitialFields:     opt.InitialFields,
	}
	return config.Build(opt.Options...)
}
