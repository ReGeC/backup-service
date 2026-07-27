// internal/scheduler/scheduler.go
package scheduler

import (
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	cronInstance *cron.Cron
	once         sync.Once
)

// Init инициализирует планировщик (singleton)
func Init() *cron.Cron {
	once.Do(func() {
		cronInstance = cron.New(cron.WithSeconds()) // с секундами для гибкости
	})
	return cronInstance
}

// Start запускает планировщик
func Start() {
	Init().Start()
	log.Println("Cron запущен")
}

// Stop останавливает планировщик (возвращает контекст для ожидания)
func Stop() {
	if cronInstance != nil {
		ctx := cronInstance.Stop()
		<-ctx.Done()
		log.Println("Cron остановлен")
	}
}

// AddJob добавляет задачу с расписанием в формате cron
// spec - строка расписания: "0 3 * * *" (каждый день в 3 часа)
// job - функция, которая будет выполняться
func AddJob(spec string, job func()) error {
	_, err := Init().AddFunc(spec, job)
	if err != nil {
		log.Printf("Ошибка добавления задачи: %v", err)
		return err
	}
	log.Printf("Задача добавлена: %s", spec)
	return nil
}

// AddJobWithID добавляет задачу и возвращает ID (для удаления)
func AddJobWithID(spec string, job func()) (cron.EntryID, error) {
	return Init().AddFunc(spec, job)
}

// RemoveJob удаляет задачу по ID
func RemoveJob(id cron.EntryID) {
	Init().Remove(id)
	log.Printf("Задача %d удалена", id)
}