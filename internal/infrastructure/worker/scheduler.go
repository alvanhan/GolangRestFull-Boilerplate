package worker

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"file-management-service/pkg/logger"
)

type Scheduler struct {
	cron   *cron.Cron
	client *Client
}

func NewScheduler(client *Client) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		client: client,
	}
}

// RegisterJobs adds all periodic background tasks to the cron scheduler.
func (s *Scheduler) RegisterJobs() {
	// cleanup expired share links every hour
	s.addJob("@every 1h", func() {
		task, err := NewExpiredLinkCleanupTask()
		if err != nil {
			logger.Error("failed to create expired link cleanup task", zap.Error(err))
			return
		}
		if err := s.client.Enqueue(task); err != nil {
			logger.Error("failed to enqueue expired link cleanup task", zap.Error(err))
		}
	})

	// cleanup expired upload chunks every 6 hours
	s.addJob("@every 6h", func() {
		task, err := NewChunkCleanupTask()
		if err != nil {
			logger.Error("failed to create chunk cleanup task", zap.Error(err))
			return
		}
		if err := s.client.Enqueue(task); err != nil {
			logger.Error("failed to enqueue chunk cleanup task", zap.Error(err))
		}
	})

	// generate storage report daily
	s.addJob("@daily", func() {
		task, err := NewStorageReportTask()
		if err != nil {
			logger.Error("failed to create storage report task", zap.Error(err))
			return
		}
		if err := s.client.Enqueue(task); err != nil {
			logger.Error("failed to enqueue storage report task", zap.Error(err))
		}
	})

	// audit log cleanup weekly — deletes logs older than 1 year
	s.addJob("@weekly", func() {
		task, err := NewAuditCleanupTask()
		if err != nil {
			logger.Error("failed to create audit cleanup task", zap.Error(err))
			return
		}
		if err := s.client.Enqueue(task); err != nil {
			logger.Error("failed to enqueue audit cleanup task", zap.Error(err))
		}
	})

	// cleanup orphaned temp files every 30 minutes
	s.addJob("@every 30m", func() {
		task, err := NewFileCleanupTask(&FileCleanupPayload{})
		if err != nil {
			logger.Error("failed to create file cleanup task", zap.Error(err))
			return
		}
		if err := s.client.Enqueue(task); err != nil {
			logger.Error("failed to enqueue file cleanup task", zap.Error(err))
		}
	})
}

func (s *Scheduler) Start() {
	s.cron.Start()
	logger.Info("scheduler started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	logger.Info("scheduler stopped")
}

func (s *Scheduler) addJob(spec string, fn func()) {
	if _, err := s.cron.AddFunc(spec, fn); err != nil {
		logger.Error("failed to register cron job", zap.String("spec", spec), zap.Error(err))
	}
}
