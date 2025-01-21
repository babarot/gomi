package main

import (
	"context"
	"log/slog"
)

type WrapHandler struct {
	handler slog.Handler
	fn      WrapFunc
}

type WrapFunc func() []slog.Attr

func NewWrapHandler(h slog.Handler, fn WrapFunc) *WrapHandler {
	if lh, ok := h.(*WrapHandler); ok {
		h = lh.Handler()
	}
	return &WrapHandler{handler: h, fn: fn}
}

func (h *WrapHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *WrapHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(h.fn()...)
	return h.handler.Handle(ctx, r)
}

func (h *WrapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewWrapHandler(h.handler.WithAttrs(attrs), h.fn)
}

func (h *WrapHandler) WithGroup(name string) slog.Handler {
	return NewWrapHandler(h.handler.WithGroup(name), h.fn)
}

func (h *WrapHandler) Handler() slog.Handler {
	return h.handler
}
