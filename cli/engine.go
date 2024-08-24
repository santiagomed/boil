package cli

import (
	"context"
	"sync"
	"time"

	"github.com/santiagomed/boil/core"
	"github.com/santiagomed/boil/fs"
	"github.com/santiagomed/boil/llm"
	"github.com/santiagomed/boil/logger"
)

type ExecutionRequest struct {
	Request    *core.Request
	ResultChan chan error
	CreatedAt  time.Time
}

type Engine struct {
	pub          core.StepPublisher
	logger       logger.Logger
	requests     chan ExecutionRequest
	workers      int
	workerWG     sync.WaitGroup
	shutdownChan chan struct{}
	fs           *fs.FileSystem
	tellmURL     string
}

func NewProjectEngine(pub core.StepPublisher, l logger.Logger, workers int, fs *fs.FileSystem, tellmURL string) (*Engine, error) {
	if l == nil {
		l = logger.NewNullLogger()
	}
	return &Engine{
		pub:          pub,
		logger:       l,
		requests:     make(chan ExecutionRequest, 1000), // Buffered channel
		workers:      workers,
		shutdownChan: make(chan struct{}),
		fs:           fs,
		tellmURL:     tellmURL,
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
			r := req.Request
			llmCfg := llm.LlmConfig{
				APIKey:    r.APIKey,
				ModelName: r.ModelName,
				BatchID:   llm.EnsureBatchID(r.ProjectName),
				TellmURL:  e.tellmURL,
			}
			llmOpenAI, err := llm.NewOpenAIClient(&llmCfg, e.logger)
			if err != nil {
				req.ResultChan <- err
				close(req.ResultChan)
				continue
			}

			stepManager := core.NewDefaultStepManager(llmOpenAI, e.fs)
			pipeline, err := core.NewPipeline(req.Request, stepManager, e.pub, e.logger)
			if err != nil {
				req.ResultChan <- err
				close(req.ResultChan)
				continue
			}
			err = pipeline.Execute(ctx)
			req.ResultChan <- err
			close(req.ResultChan)
		case <-ctx.Done():
			return
		case <-e.shutdownChan:
			return
		}
	}
}

func (e *Engine) AddRequest(request *core.Request) chan error {
	resultChan := make(chan error, 1)
	e.requests <- ExecutionRequest{
		Request:    request,
		ResultChan: resultChan,
		CreatedAt:  time.Now(),
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
		e.logger.Info("All workers shut down gracefully")
	case <-time.After(timeout):
		e.logger.Warn("Shutdown timed out, some workers may still be running")
	}
}
