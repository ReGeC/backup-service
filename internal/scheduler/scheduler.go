package scheduler

import (
	"context"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

// Scheduler - планировщик задач на основе cron
type Scheduler struct {
	cron *cron.Cron
	mu   sync.Mutex
}

// New создает новый экземпляр планировщика
func New() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
	}
}

// Start запускает планировщик
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("Планировщик cron запущен")
}

// Stop останавливает планировщик и ожидает завершения текущих задач
func (s *Scheduler) Stop(ctx context.Context) {
	if s.cron == nil {
		return
	}
	
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
	log.Println("Планировщик cron остановлен")
}

// AddJob добавляет задачу с расписанием
// spec - строка расписания в формате cron (например: "0 3 * * *")
// job - функция, которая будет выполняться
// Возвращает ID задачи для возможного удаления
func (s *Scheduler) AddJob(spec string, job func()) (cron.EntryID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	id, err := s.cron.AddFunc(spec, job)
	if err != nil {
		log.Printf("Ошибка добавления задачи с расписанием %s: %v", spec, err)
		return id, err
	}
	
	log.Printf("Задача добавлена с расписанием: %s (ID: %d)", spec, id)
	return id, nil
}

// RemoveJob удаляет задачу по ID
func (s *Scheduler) RemoveJob(id cron.EntryID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cron.Remove(id)
	log.Printf("Задача с ID %d удалена", id)
}

// GetEntries возвращает список всех задач
func (s *Scheduler) GetEntries() []cron.Entry {
	return s.cron.Entries()
}