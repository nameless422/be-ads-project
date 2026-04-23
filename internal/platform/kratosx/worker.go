package kratosx

import (
	"context"
	"log"
)

type WorkerServer struct {
	name   string
	logger *log.Logger
	run    func(context.Context) error
	cancel context.CancelFunc
}

func NewWorkerServer(name string, logger *log.Logger, run func(context.Context) error) *WorkerServer {
	return &WorkerServer{
		name:   name,
		logger: logger,
		run:    run,
	}
}

func (s *WorkerServer) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.logger.Printf("[%s] kratos worker start", s.name)
	return s.run(runCtx)
}

func (s *WorkerServer) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	s.logger.Printf("[%s] kratos worker stop requested", s.name)
	return nil
}
