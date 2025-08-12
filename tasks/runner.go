package tasks

import (
	"context"
	"log"
	"time"
)

type IService interface {
	ClaimDue(ctx context.Context) (*Task, error)
	MarkFailed(ctx context.Context, ID string) error
}

type Handler func(ctx context.Context, t Task) error

type Runner struct {
	service  IService
	handlers map[string]Handler
}

func NewRunner(service IService, handlers map[string]Handler) *Runner {
	return &Runner{
		service:  service,
		handlers: handlers,
	}
}

func (r *Runner) Run(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {

		go func() {
			ticker := time.NewTicker(3 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					t, err := r.service.ClaimDue(ctx)
					if err != nil {
						continue
					}
					h := r.handlers[t.Type]
					if h == nil {
						_ = r.service.MarkFailed(ctx, t.ID)
						continue
					}
					if err := h(ctx, *t); err != nil {
						_ = r.service.MarkFailed(ctx, t.ID)
						log.Printf("task %s failed: %v", t.Type, err)
						continue
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}
