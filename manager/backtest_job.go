package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	JobPending     = "pending"
	JobDownloading = "downloading"
	JobRunning     = "running"
	JobCompleted   = "completed"
	JobFailed      = "failed"
)

type BacktestJob struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	Strategy      string          `json:"strategy"`
	Config        json.RawMessage `json:"config"`
	Exchange      string          `json:"exchange"`
	Symbol        string          `json:"symbol"`
	StartTime     string          `json:"start_time"`
	EndTime       string          `json:"end_time"`
	Status        string          `json:"status"`
	Progress      string          `json:"progress,omitempty"`
	Output        string          `json:"output,omitempty"`
	Report        json.RawMessage `json:"report,omitempty"`
	EquityCurve   string          `json:"equity_curve,omitempty"`
	Error         string          `json:"error,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	NeedSync      bool            `json:"need_sync"`
	FuturesConfig *FuturesConfig  `json:"futuresConfig,omitempty"`
}

type BacktestJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*BacktestJob
	dir  string
	sem  chan struct{}
}

func NewBacktestJobStore(dataDir string) *BacktestJobStore {
	dir := filepath.Join(dataDir, "backtest-jobs")
	os.MkdirAll(dir, 0o755)

	s := &BacktestJobStore{
		jobs: make(map[string]*BacktestJob),
		dir:  dir,
		sem:  make(chan struct{}, 2),
	}

	s.loadPersisted()
	return s
}

func (s *BacktestJobStore) Create(job *BacktestJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.CreatedAt = time.Now()
	job.Status = JobPending
	s.jobs[job.ID] = job
	s.persist(job)
}

func (s *BacktestJobStore) Get(jobID string) (*BacktestJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[jobID]
	if !ok {
		return nil, false
	}
	cp := *j
	return &cp, true
}

func (s *BacktestJobStore) ListByUser(userID string) []*BacktestJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*BacktestJob
	for _, j := range s.jobs {
		if j.UserID == userID {
			cp := *j
			result = append(result, &cp)
		}
	}
	return result
}

func (s *BacktestJobStore) UpdateStatus(jobID, status, progress string) *BacktestJob {
	return s.updateStatus(jobID, status, progress, "")
}

func (s *BacktestJobStore) FailJob(jobID, progress, errMsg string) *BacktestJob {
	return s.updateStatus(jobID, JobFailed, progress, errMsg)
}

func (s *BacktestJobStore) updateStatus(jobID, status, progress, errMsg string) *BacktestJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[jobID]
	if !ok {
		return nil
	}
	j.Status = status
	j.Progress = progress
	if errMsg != "" {
		j.Error = errMsg
	}
	now := time.Now()
	if status == JobDownloading || status == JobRunning {
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
	}
	if status == JobCompleted || status == JobFailed {
		j.CompletedAt = &now
	}
	s.persist(j)
	cp := *j
	return &cp
}

func (s *BacktestJobStore) SetOutput(jobID, output string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[jobID]; ok {
		j.Output = output
		s.persist(j)
	}
}

func (s *BacktestJobStore) SetError(jobID, err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[jobID]; ok {
		j.Error = err
		s.persist(j)
	}
}

func (s *BacktestJobStore) SetReport(jobID string, report json.RawMessage, equityCurve string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[jobID]; ok {
		j.Report = report
		j.EquityCurve = equityCurve
		s.persist(j)
	}
}

func (s *BacktestJobStore) AcquireSlot() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *BacktestJobStore) ReleaseSlot() {
	<-s.sem
}

func (s *BacktestJobStore) persist(job *BacktestJob) {
	path := s.jobPath(job.ID)
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		log.Printf("persist backtest job %s: %v", job.ID, err)
		return
	}
	os.WriteFile(path, data, 0o600)
}

func (s *BacktestJobStore) jobPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *BacktestJobStore) loadPersisted() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var job BacktestJob
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}
		if job.Status == JobDownloading || job.Status == JobRunning || job.Status == JobPending {
			if job.StartedAt != nil && !job.StartedAt.IsZero() && time.Since(*job.StartedAt) > 5*time.Minute {
				job.Status = JobFailed
				job.Progress = "job interrupted by manager restart"
				s.persist(&job)
			} else {
				job.Status = JobPending
				job.Progress = ""
			}
		}
		s.jobs[job.ID] = &job
	}
	log.Printf("loaded %d persisted backtest jobs", len(s.jobs))
	s.Prune(24*time.Hour, nil)
	log.Printf("after prune: %d backtest jobs remaining", len(s.jobs))
}

func (s *BacktestJobStore) Prune(olderThan time.Duration, storage *StorageClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	for id, j := range s.jobs {
		if j.Status == JobCompleted || j.Status == JobFailed {
			if j.CompletedAt != nil && j.CompletedAt.Before(cutoff) {
				if storage != nil {
					storage.RemoveFolder(j.UserID, id)
				}
				delete(s.jobs, id)
				os.Remove(s.jobPath(id))
			}
		}
	}
}

type BacktestExecutor struct {
	store     *BacktestJobStore
	container *ContainerManager
	notifier  *Notifier
	storage   *StorageClient
	defaults  DefaultsProvider

	// Test hooks — override in tests to mock Docker operations.
	syncFn    func(userID, exchange, symbol, startTime, endTime string) (string, error)
	runFn     func(userID string, jobID string, yamlContent []byte) ([]byte, error)
	reportFn  func(userID, jobID string) (json.RawMessage, []byte, error)
}

func NewBacktestExecutor(store *BacktestJobStore, cm *ContainerManager, notifier *Notifier, storage *StorageClient, defaults DefaultsProvider) *BacktestExecutor {
	return &BacktestExecutor{
		store:     store,
		container: cm,
		notifier:  notifier,
		storage:   storage,
		defaults:  defaults,
	}
}

func (ex *BacktestExecutor) syncBacktest(userID, exchange, symbol, startTime, endTime string) (string, error) {
	if ex.syncFn != nil {
		return ex.syncFn(userID, exchange, symbol, startTime, endTime)
	}
	return ex.container.SyncBacktest(userID, exchange, symbol, startTime, endTime)
}

func (ex *BacktestExecutor) runBacktest(userID string, jobID string, yamlContent []byte) ([]byte, error) {
	if ex.runFn != nil {
		return ex.runFn(userID, jobID, yamlContent)
	}
	return ex.container.RunBacktest(userID, jobID, yamlContent)
}

func (ex *BacktestExecutor) Submit(job *BacktestJob) error {
	ex.store.Create(job)

	if !ex.store.AcquireSlot() {
		ex.store.FailJob(job.ID, "too many concurrent backtest jobs", "server busy, try again later")
		return fmt.Errorf("server busy")
	}

	go ex.execute(job)
	return nil
}

func (ex *BacktestExecutor) execute(job *BacktestJob) {
	defer ex.store.ReleaseSlot()

	if job.NeedSync {
		ex.store.UpdateStatus(job.ID, JobDownloading, "syncing market data...")
		out, err := ex.syncBacktest(job.UserID, job.Exchange, job.Symbol, job.StartTime, job.EndTime)
		if err != nil {
			ex.store.FailJob(job.ID, "data sync failed", fmt.Sprintf("data sync failed: %s", err))
			ex.notify(job, "Backtest Data Sync Failed", fmt.Sprintf("Strategy %s on %s/%s: data sync failed", job.Strategy, job.Exchange, job.Symbol))
			return
		}
		log.Printf("backtest data synced for job %s: %s", job.ID, out)
	}

	ex.store.UpdateStatus(job.ID, JobRunning, "running backtest...")

	yamlContent, err := buildBacktestYAML(job.Strategy, job.Config, job.StartTime, job.EndTime, job.Exchange, job.Symbol, ex.defaults, job.FuturesConfig)
	if err != nil {
		ex.store.FailJob(job.ID, "config error", fmt.Sprintf("invalid config: %v", err))
		return
	}

	result, err := ex.runBacktest(job.UserID, job.ID, yamlContent)
	if err != nil {
		if ex.storage != nil {
		ex.container.CleanupBacktest(job.UserID, job.ID)
	}
		ex.store.FailJob(job.ID, "backtest failed", err.Error())
		ex.notify(job, "Backtest Failed", fmt.Sprintf("Strategy %s: %s", job.Strategy, err.Error()))
		return
	}

	ex.store.SetOutput(job.ID, string(result))

	report, equityCurve, reportErr := ex.readReport(job.UserID, job.ID)
	if reportErr != nil {
		log.Printf("backtest report read for job %s: %v (non-fatal)", job.ID, reportErr)
	} else {
		ex.store.SetReport(job.ID, report, string(equityCurve))
		ex.uploadToStorage(job.UserID, job.ID, report, equityCurve)
	}

	ex.store.UpdateStatus(job.ID, JobCompleted, "done")
	ex.notify(job, "Backtest Completed", fmt.Sprintf("Strategy %s on %s/%s completed successfully", job.Strategy, job.Exchange, job.Symbol))
}

func (ex *BacktestExecutor) notify(job *BacktestJob, title, message string) {
	if ex.notifier == nil {
		return
	}
	ex.notifier.Dispatch(job.UserID, NotificationEvent{
		Type:    "backtest",
		Title:   title,
		Message: message,
	})
}

func (ex *BacktestExecutor) readReport(userID, jobID string) (json.RawMessage, []byte, error) {
	if ex.reportFn != nil {
		return ex.reportFn(userID, jobID)
	}
	if ex.container == nil {
		return nil, nil, fmt.Errorf("no container manager")
	}
	return ex.container.ReadBacktestReport(userID, jobID)
}

func (ex *BacktestExecutor) uploadToStorage(userID, jobID string, report json.RawMessage, equityCurve []byte) {
	if ex.storage == nil {
		return
	}
	if err := ex.storage.Upload(userID, jobID, "summary.json", report); err != nil {
		log.Printf("storage upload summary for job %s: %v", jobID, err)
	}
	if len(equityCurve) > 0 {
		if err := ex.storage.Upload(userID, jobID, "equity_curve.tsv", equityCurve); err != nil {
			log.Printf("storage upload equity for job %s: %v", jobID, err)
		}
	}
	reportDir := ex.container.BacktestReportDir(userID, jobID)
	for _, name := range []string{"trades.tsv", "orders.tsv"} {
		data, err := os.ReadFile(filepath.Join(reportDir, name))
		if err != nil {
			continue
		}
		if err := ex.storage.Upload(userID, jobID, name, data); err != nil {
			log.Printf("storage upload %s for job %s: %v", name, jobID, err)
		}
	}
	// Upload kline files (e.g., BTCUSDT-1h.tsv)
	if matches, err := filepath.Glob(filepath.Join(reportDir, "*-*.tsv")); err == nil {
		for _, match := range matches {
			name := filepath.Base(match)
			if name == "trades.tsv" || name == "orders.tsv" || name == "equity_curve.tsv" {
				continue
			}
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			if err := ex.storage.Upload(userID, jobID, name, data); err != nil {
				log.Printf("storage upload %s for job %s: %v", name, jobID, err)
			}
		}
	}
}
