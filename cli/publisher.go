package cli

import (
	"fmt"

	"github.com/santiagomed/boil/core"
	"github.com/santiagomed/boil/logger"
)

type CliStepPublisher struct {
	stepChan  chan core.StepType
	errorChan chan error
	logger    logger.Logger
}

func NewCliStepPublisher(logger logger.Logger) *CliStepPublisher {
	return &CliStepPublisher{
		stepChan:  make(chan core.StepType, 100), // Buffer size of 100
		errorChan: make(chan error, 10),          // Buffer size of 10
		logger:    logger,
	}
}

func (p *CliStepPublisher) PublishStep(step core.StepType) {
	select {
	case p.stepChan <- step:
		p.logger.Debug(fmt.Sprintf("Successfully published step: %v", step))
	default:
		p.logger.Warn(fmt.Sprintf("Failed to publish step: %v. Channel full.", step))
	}
}

func (p *CliStepPublisher) Error(step core.StepType, err error) {
	select {
	case p.errorChan <- err:
		p.logger.Debug(fmt.Sprintf("Successfully published error for step: %v", step))
	default:
		p.logger.Warn(fmt.Sprintf("Failed to publish error for step: %v. Channel full.", step))
	}
}
