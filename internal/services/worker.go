package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"alfredoptarigan/cv-evaluator/internal/repositories"
)

type Worker interface {
	Start(ctx context.Context)
	Stop()
	EnqueueJob(evalID uuid.UUID)
}

type worker struct {
	evalRepo         repositories.EvaluationRepository
	evaluatorService EvaluatorService
	jobQueue         chan uuid.UUID
	concurrency      int
	wg               sync.WaitGroup
	stopChan         chan struct{}
}

func NewWorker(
	evalRepo repositories.EvaluationRepository,
	evaluatorService EvaluatorService,
	concurrency int,
) Worker {
	return &worker{
		evalRepo:         evalRepo,
		evaluatorService: evaluatorService,
		jobQueue:         make(chan uuid.UUID, 100),
		concurrency:      concurrency,
		stopChan:         make(chan struct{}),
	}
}

// Start implements Worker.
func (w *worker) Start(ctx context.Context) {
	log.Printf("🚀 Starting worker with %d concurrent workers\n", w.concurrency)

	// Start worker goroutines
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.processJobs(ctx, i+1)
	}

	// Start polling for pending jobs
	w.wg.Add(1)
	go w.pollPendingJobs(ctx)

	log.Println("✅ Worker started successfully")
}

// Stop implements Worker.
func (w *worker) Stop() {
	log.Println("🛑 Stopping worker...")
	close(w.stopChan)
	w.wg.Wait()
	log.Println("✅ Worker stopped")
}

// EnqueueJob implements Worker.
func (w *worker) EnqueueJob(evalID uuid.UUID) {
	select {
	case w.jobQueue <- evalID:
		log.Printf("📥 Job %s enqueued\n", evalID)
	case <-w.stopChan:
		log.Printf("⚠️  Worker stopped, cannot enqueue job %s\n", evalID)
	}
}

func (w *worker) processJobs(ctx context.Context, workerID int) {
	defer w.wg.Done()
	log.Printf("🚀 Worker %d started processing jobs\n", workerID)

	for {
		select {
		case <-w.stopChan:
			log.Printf("👷 Worker #%d stopped\n", workerID)
			return
		case evalID := <-w.jobQueue:
			log.Printf("👷 Worker #%d processing job %s\n", workerID, evalID)
			// Process the evaluation
			if err := w.evaluatorService.EvaluateCandidate(ctx, evalID); err != nil {
				log.Printf("❌ Worker #%d failed to process job %s: %v\n", workerID, evalID, err)
			} else {
				log.Printf("✅ Worker #%d completed job %s\n", workerID, evalID)
			}
		}
	}
}

func (w *worker) pollPendingJobs(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Println("🔄 Starting pending jobs poller")

	for {
		select {
		case <-w.stopChan:
			log.Println("🔄 Pending jobs poller stopped")
			return
		case <-ticker.C:
			// Find pending jobs
			pendingJobs, err := w.evalRepo.FindPendingJobs(10)
			if err != nil {
				log.Printf("⚠️  Failed to fetch pending jobs: %v\n", err)
				continue
			}

			if len(pendingJobs) > 0 {
				log.Printf("📋 Found %d pending jobs\n", len(pendingJobs))
			}

			// Enqueue pending jobs
			for _, job := range pendingJobs {
				w.EnqueueJob(job.ID)
			}
		}
	}
}
