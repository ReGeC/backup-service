package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New()
	assert.NotNil(t, s)
	assert.NotNil(t, s.cron)
}

func TestScheduler_AddJob(t *testing.T) {
	s := New()
	
	tests := []struct {
		name    string
		spec    string
		job     func()
		wantErr bool
	}{
		{
			name:    "валидное расписание",
			spec:    "0 0 3 * * *",
			job:     func() {},
			wantErr: false,
		},
		{
			name:    "каждую минуту",
			spec:    "0 * * * * *",
			job:     func() {},
			wantErr: false,
		},
		{
			name:    "каждую секунду",
			spec:    "*/1 * * * * *",
			job:     func() {},
			wantErr: false,
		},
		{
			name:    "невалидное расписание (5 полей)",
			spec:    "0 3 * * *",
			job:     func() {},
			wantErr: true,
		},
		{
			name:    "невалидное расписание",
			spec:    "invalid",
			job:     func() {},
			wantErr: true,
		},
		{
			name:    "пустое расписание",
			spec:    "",
			job:     func() {},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := s.AddJob(tt.spec, tt.job)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, cron.EntryID(0), id)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, cron.EntryID(0), id)
			}
		})
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	s := New()
	
	// Запускаем
	s.Start()
	
	// Проверяем, что задачи можно добавлять после старта
	_, err := s.AddJob("0 0 3 * * *", func() {})
	assert.NoError(t, err)
	
	// Останавливаем
	ctx := context.Background()
	s.Stop(ctx)
	
	// Проверяем, что после остановки задачи не выполняются
	counter := int32(0)
	_, err = s.AddJob("*/1 * * * * *", func() { atomic.AddInt32(&counter, 1) })
	assert.NoError(t, err) // задача добавится, но выполняться не будет
	
	time.Sleep(2 * time.Second)
	assert.Equal(t, int32(0), atomic.LoadInt32(&counter))
}

func TestScheduler_AddAndRemoveJob(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	// Добавляем задачу
	counter := int32(0)
	job := func() {
		atomic.AddInt32(&counter, 1)
	}
	
	id, err := s.AddJob("*/1 * * * * *", job) // каждую секунду
	require.NoError(t, err)
	require.NotEqual(t, cron.EntryID(0), id)
	
	// Проверяем, что задача есть в списке
	entries := s.GetEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, id, entries[0].ID)
	
	// Ждем выполнения
	time.Sleep(2 * time.Second)
	assert.Greater(t, atomic.LoadInt32(&counter), int32(0))
	
	// Удаляем задачу
	s.RemoveJob(id)
	
	// Проверяем, что задача удалена
	entries = s.GetEntries()
	assert.Len(t, entries, 0)
	
	// Ждем еще и проверяем, что счетчик не увеличился
	oldCounter := atomic.LoadInt32(&counter)
	time.Sleep(2 * time.Second)
	assert.Equal(t, oldCounter, atomic.LoadInt32(&counter))
}

func TestScheduler_MultipleJobs(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	counter1 := int32(0)
	counter2 := int32(0)
	
	_, err := s.AddJob("*/1 * * * * *", func() { atomic.AddInt32(&counter1, 1) })
	require.NoError(t, err)
	
	_, err = s.AddJob("*/2 * * * * *", func() { atomic.AddInt32(&counter2, 1) })
	require.NoError(t, err)
	
	// Проверяем, что обе задачи добавлены
	entries := s.GetEntries()
	assert.Len(t, entries, 2)
	
	// Ждем выполнения
	time.Sleep(3 * time.Second)
	
	assert.Greater(t, atomic.LoadInt32(&counter1), int32(0))
	assert.Greater(t, atomic.LoadInt32(&counter2), int32(0))
}

func TestScheduler_StopWithCancel(t *testing.T) {
	s := New()
	s.Start()
	
	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	
	// Добавляем задачу
	jobExecuted := int32(0)
	
	_, err := s.AddJob("*/1 * * * * *", func() {
		atomic.AddInt32(&jobExecuted, 1)
	})
	require.NoError(t, err)
	
	// Ждем выполнения
	time.Sleep(2 * time.Second)
	assert.Greater(t, atomic.LoadInt32(&jobExecuted), int32(0))
	
	// Отменяем контекст и останавливаем планировщик
	cancel()
	s.Stop(ctx)
	
	// Запоминаем текущее значение
	oldCounter := atomic.LoadInt32(&jobExecuted)
	
	// Ждем и проверяем, что счетчик не увеличился
	time.Sleep(2 * time.Second)
	assert.Equal(t, oldCounter, atomic.LoadInt32(&jobExecuted))
}

func TestScheduler_ConcurrentAddAndRemove(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	// Добавляем и удаляем задачи конкурентно
	done := make(chan bool)
	
	go func() {
		for i := 0; i < 10; i++ {
			id, err := s.AddJob("*/1 * * * * *", func() {})
			if err == nil {
				s.RemoveJob(id)
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 10; i++ {
			_ = s.GetEntries()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()
	
	// Ждем завершения всех горутин
	<-done
	<-done
	
	// Проверяем, что планировщик все еще жив - можно добавить задачу
	_, err := s.AddJob("0 0 3 * * *", func() {})
	assert.NoError(t, err)
}

func TestScheduler_StopWhenNotStarted(t *testing.T) {
	s := New()
	
	// Останавливаем без запуска
	ctx := context.Background()
	s.Stop(ctx)
	
	// Проверяем, что ничего не сломалось
	_, err := s.AddJob("0 0 3 * * *", func() {})
	assert.NoError(t, err)
}

func TestScheduler_GetEntries_Empty(t *testing.T) {
	s := New()
	
	entries := s.GetEntries()
	assert.Empty(t, entries)
}

func TestScheduler_AddJobWithSameSchedule(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	counter1 := int32(0)
	counter2 := int32(0)
	
	_, err := s.AddJob("*/1 * * * * *", func() { atomic.AddInt32(&counter1, 1) })
	require.NoError(t, err)
	
	_, err = s.AddJob("*/1 * * * * *", func() { atomic.AddInt32(&counter2, 1) })
	require.NoError(t, err)
	
	// Проверяем, что обе задачи добавлены
	entries := s.GetEntries()
	assert.Len(t, entries, 2)
	
	// Ждем выполнения
	time.Sleep(2 * time.Second)
	
	assert.Greater(t, atomic.LoadInt32(&counter1), int32(0))
	assert.Greater(t, atomic.LoadInt32(&counter2), int32(0))
}

func TestScheduler_MultipleStop(t *testing.T) {
	s := New()
	s.Start()
	
	ctx := context.Background()
	
	// Первая остановка
	s.Stop(ctx)
	
	// Вторая остановка (должна быть безопасной)
	s.Stop(ctx)
	
	// Проверяем, что можно добавить задачу после остановки
	_, err := s.AddJob("0 0 3 * * *", func() {})
	assert.NoError(t, err)
}

func TestScheduler_JobExecution(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	counter := int32(0)
	
	// Добавляем задачу, которая выполняется каждую секунду
	_, err := s.AddJob("*/1 * * * * *", func() {
		atomic.AddInt32(&counter, 1)
	})
	require.NoError(t, err)
	
	// Ждем 3 секунды, должно быть примерно 3 выполнения
	time.Sleep(3 * time.Second)
	
	counterValue := atomic.LoadInt32(&counter)
	assert.GreaterOrEqual(t, counterValue, int32(2))
	assert.LessOrEqual(t, counterValue, int32(5)) // допустимый разброс
}

// Бенчмарки
func BenchmarkScheduler_AddJob(b *testing.B) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	for b.Loop() {
		_, err := s.AddJob("0 0 3 * * *", func() {})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScheduler_GetEntries(b *testing.B) {
	s := New()
	s.Start()
	defer s.Stop(context.Background())
	
	// Добавляем 10 задач
	for i := 0; i < 10; i++ {
		_, _ = s.AddJob("0 0 3 * * *", func() {})
	}
	
	b.ResetTimer()
	for b.Loop() {
		_ = s.GetEntries()
	}
}