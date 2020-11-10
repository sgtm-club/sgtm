package sgtm

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/banner"
	"moul.io/sgtm/pkg/sgtmpb"
)

type processingWorkerDriver struct {
	started bool
	wg      *sync.WaitGroup

	trackMigrations []func(*sgtmpb.Post) error
}

func (svc *Service) StartProcessingWorker() error {
	// init
	{
		fmt.Fprintln(os.Stderr, banner.Inline("processing-worker"))
		svc.logger.Debug("starting processing-worker")
		svc.setupMigrations()
		svc.processingWorker.wg = &sync.WaitGroup{}
		svc.processingWorker.wg.Add(1)
		defer svc.processingWorker.wg.Done()
		svc.processingWorker.started = true
	}

	// loop
	for i := 0; ; i++ {
		if err := svc.processingLoop(i); err != nil {
			return err
		}

		select {
		// FIXME: add a channel to get instant worker task
		case <-time.After(30 * time.Second):
		case <-svc.ctx.Done():
			return nil
		}
	}
}

func (svc *Service) CloseProcessingWorker(err error) {
	svc.logger.Debug("closing processingWorker", zap.Bool("was-started", svc.processingWorker.started), zap.Error(err))
	svc.cancel()
	if svc.processingWorker.started {
		svc.processingWorker.wg.Wait()
		svc.logger.Debug("processing-worker closed")
	}
}

func (svc *Service) processingLoop(i int) error {
	before := time.Now()

	// track migrations
	{
		var outdated []*sgtmpb.Post
		err := svc.rodb().
			Where(sgtmpb.Post{Kind: sgtmpb.Post_TrackKind}).
			Where("processing_error IS NULL OR processing_error == ''").
			Where("processing_version IS NULL OR processing_version < ?", len(svc.processingWorker.trackMigrations)).
			Preload("Author").
			Find(&outdated).
			Error
		if err != nil {
			return fmt.Errorf("failed to fetch tracks that need to be processed: %w", err)
		}

		err = svc.rwdb().Transaction(func(db *gorm.DB) error {
			for _, entryPtr := range outdated {
				entry := entryPtr
				version := 1
				for _, migration := range svc.processingWorker.trackMigrations {
					err := migration(entry)
					if err != nil {
						entry.ProcessingError = err.Error()
						break
					}
					entry.ProcessingVersion = int64(version)
					version++
				}
				if err := db.
					Model(&entry).
					Updates(map[string]interface{}{
						"processing_version": entry.ProcessingVersion,
						"processing_error":   entry.ProcessingError,
					}).
					Error; err != nil {
					return fmt.Errorf("failed to save processing state: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	// TODO: other type migrations
	// TODO: track maintenance (i.e., daily check if the track still exists on SoundCloud)

	svc.logger.Debug("processing loop ended",
		zap.Duration("duration", time.Since(before)),
		zap.Int("loop", i),
	)
	return nil
}

func (svc *Service) setupMigrations() {
	svc.processingWorker.trackMigrations = []func(*sgtmpb.Post) error{
		/*
			// FIXME: try downloading the mp3 locally
			func(post *sgtmpb.Post) error { return fmt.Errorf("not implemented") },
			// FIXME: compute BPM
			func(post *sgtmpb.Post) error { return fmt.Errorf("not implemented") },
			// FIXME: extract thumbnail from file metadata
			func(post *sgtmpb.Post) error { return fmt.Errorf("not implemented") },
			// FIXME: compute other info with analysis tools
			func(post *sgtmpb.Post) error { return fmt.Errorf("not implemented") },
			// FIXME: create MP3 version for uploaded WAV
			func(post *sgtmpb.Post) error {
				if post.Provider != sgtmpb.Provider_IPFS {
					return nil
				}
				// if post.mp3_192_cid == "" && format != mp3 { download; compress; upload }
				return nil
			},
		*/
	}
}
