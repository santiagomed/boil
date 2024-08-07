package core

import (
	"context"
	"sync"
	"time"

	"github.com/santiagomed/boil/pkg/config"

	"github.com/rs/zerolog"
)

type Request struct {
	ProjectDesc string
	ResultChan  chan error
	CreatedAt   time.Time
}

type Engine struct {
	config       *config.Config
	pub          StepPublisher
	logger       *zerolog.Logger
	requests     chan Request
	workers      int
	workerWG     sync.WaitGroup
	shutdownChan chan struct{}
}

func NewProjectEngine(config *config.Config, pub StepPublisher, logger *zerolog.Logger, workers int) (*Engine, error) {
	return &Engine{
		config:       config,
		pub:          pub,
		logger:       logger,
		requests:     make(chan Request, 1000), // Buffered channel
		workers:      workers,
		shutdownChan: make(chan struct{}),
	}, nil
}

func (e *Engine) Start(ctx context.Context) {
	for i := 0; i < e.workers; i++ {
		e.workerWG.Add(1)
		go e.worker(ctx)
	}
}

func (e *Engine) worker(ctx context.Context) {
	defer e.workerWG.Done()
	for {
		select {
		case req := <-e.requests:
			pipeline, err := NewPipeline(e.config, e.pub, e.logger)
			if err != nil {
				req.ResultChan <- err
				close(req.ResultChan)
				continue
			}
			err = pipeline.Execute(req.ProjectDesc)
			req.ResultChan <- err
			close(req.ResultChan)
		case <-ctx.Done():
			return
		case <-e.shutdownChan:
			return
		}
	}
}

func (e *Engine) AddRequest(projectDesc string) chan error {
	resultChan := make(chan error, 1)
	e.requests <- Request{
		ProjectDesc: projectDesc,
		ResultChan:  resultChan,
		CreatedAt:   time.Now(),
	}
	return resultChan
}

func (e *Engine) Shutdown(timeout time.Duration) {
	close(e.shutdownChan)

	done := make(chan struct{})
	go func() {
		e.workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info().Msg("All workers shut down gracefully")
	case <-time.After(timeout):
		e.logger.Warn().Msg("Shutdown timed out, some workers may still be running")
	}
}
