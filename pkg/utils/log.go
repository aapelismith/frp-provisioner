package utils

import (
	"context"
	"go.uber.org/zap"
)

type loggerKey struct{}

// WithContext Inject logger into the context
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	if ctx == nil {
		panic("nil context")
	}

	if logger == nil {
		panic("nil logger")
	}

	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext get logger from context
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		panic("nil context")
	}
	logger, ok := ctx.Value(loggerKey{}).(*zap.Logger)
	if !ok {
		logger, _ = zap.NewDevelopment()
	}
	return logger
}
