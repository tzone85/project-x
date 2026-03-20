package main

import (
	"context"

	"github.com/tzone85/project-x/internal/config"
)

type contextKey int

const (
	configKey  contextKey = iota
	cleanupKey
)

func withConfig(ctx context.Context, cfg config.Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

func ConfigFromContext(ctx context.Context) (config.Config, bool) {
	cfg, ok := ctx.Value(configKey).(config.Config)
	return cfg, ok
}

func withCleanup(ctx context.Context, fn func()) context.Context {
	return context.WithValue(ctx, cleanupKey, fn)
}

func cleanupFromContext(ctx context.Context) (func(), bool) {
	fn, ok := ctx.Value(cleanupKey).(func())
	return fn, ok
}
