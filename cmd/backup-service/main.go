package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backup-service/internal/backup"
	"backup-service/internal/config"
	"backup-service/internal/notifier"
	"backup-service/internal/scheduler"
	"backup-service/internal/storage"
	"backup-service/internal/storage/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println(".env файл не найден, используются переменные по умолчанию")
	}

	cfg, err := config.NewBackupConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурационного файла: ", err)
	}

	db, err := database.InitDB("backup_service.db")
	if err != nil {
		log.Fatal("Ошибка инициализации БД: ", err)
	}

	log.Println("Сервис Бекапов запущен")

	// Инициализация хранилища
	st, err := storage.NewStorage(cfg.StorageType)
	if err != nil {
		log.Fatal("Ошибка создания хранилища: ", err)
	}

	//Инициализация уведомлений
	notifiers := notifier.InitNotifiers()

	// Инициализация бэкапов
	backuppers := backup.InitBackuppers()

	backupLogRepository := database.NewGormBackupLogRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Проверка команд и их первый запуск
	go func() {
		log.Println("Стартовый бэкап (проверка)")
		startCtx, startCancel := context.WithTimeout(ctx, 30*time.Minute)
    	defer startCancel()
		runBackupCycle(startCtx, backuppers, st, backupLogRepository, cfg, notifiers)

		log.Println("Стартовая очистка старых бэкапов (проверка)")
		cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cleanupCancel()
		cleanupOldBackups(cleanupCtx, st, notifiers, cfg.RetentionDays)
	}()
	time.Sleep(1 * time.Second)
	
	// Планировщик
	if cfg.CronEnable {
		// Создаем планировщик
		sched := scheduler.New()

		//--------------------Определяем функцию бэкапа
		backupJob := func() {
			// Создаем контекст с таймаутом для этого запуска
			jobCtx, jobCancel := context.WithTimeout(ctx, 30*time.Minute)
			defer jobCancel()

			runBackupCycle(jobCtx, backuppers, st, backupLogRepository, cfg, notifiers)
		}

		if _, err := sched.AddJob(cfg.BackupSchedule, backupJob); err != nil {
			log.Fatal("Ошибка добавления задачи backupJob в планировщик: ", err)
		}

		log.Printf("Расписание: %s", cfg.BackupSchedule)
		//-----------------------------------------

		//-------Определяем функцию очистки старых бэкапов
		cleanupJob := func() {
			// Создаем контекст с таймаутом для этого запуска
			jobCtx, jobCancel := context.WithTimeout(ctx, 1*time.Minute)
			defer jobCancel()

			cleanupOldBackups(jobCtx, st, notifiers, cfg.RetentionDays)
		}

		if _, err := sched.AddJob(getCronStringForCleanupFunc(), cleanupJob); err != nil {
			log.Fatal("Ошибка добавления задачи cleanupJob в планировщик: ", err)
		}
		//----------------------------------------------

		// Запускаем планировщик
		sched.Start()

		// Останавливаем планировщик
		defer sched.Stop(ctx)

		// Блокируем main и ждем сигнала остановки
		waitForShutdown(cancel)


		log.Println("Остановка сервиса...")
	} else {
		log.Println("Крон отключен (CRON_ENABLE=false)")
	}
	
	log.Println("Сервис остановлен")
}

// runBackupCycle выполняет полный цикл бэкапа для всех типов
func runBackupCycle(
	ctx context.Context,
	backuppers map[string]backup.Backupper,
	st storage.Storage,
	repo *database.GormBackupLogRepository,
	cfg *config.BackupConfig,
	notifiers map[string]notifier.Notifier,
) {
	// Проверяем, не отменен ли контекст
	select {
	case <-ctx.Done():
		log.Println("Бэкап пропущен: контекст отменен")
		return
	default:
	}

	log.Println("Начало цикла бэкапов")

	for typ, backupper := range backuppers {
		// Проверяем контекст перед каждым бэкапом
		select {
		case <-ctx.Done():
			log.Printf("Бэкап %s прерван: %v", typ, ctx.Err())
			msg := "Бэкап " + typ + " прерван при остановке сервиса"
			notifier.SendAll(notifiers, ctx, msg)
			return
		default:
		}

		log.Printf("Запуск бэкапа: %s", typ)

		// Создание бэкапа
		localPath, err := backup.RunBackup(ctx, backupper, repo, cfg.BackupPath, cfg.StorageType)
		if err != nil {
			msg := "Ошибка создания бэкапа " + typ + ": " + err.Error()
			notifier.SendAll(notifiers, ctx, msg)
			continue
		}

		// Сохранение в хранилище
		remotePath, err := st.Save(ctx, localPath)
		if err != nil {
			msg := "Ошибка сохранения бэкапа " + typ + " в хранилище: " + err.Error()
			notifier.SendAll(notifiers, ctx, msg)
			continue
		}

		// Уведомление об успехе
		msg := "Бэкап " + typ + " сохранен: " + remotePath
		notifier.SendAll(notifiers, ctx, msg)

		log.Printf("Бэкап %s сохранен в хранилище: %s", typ, remotePath)
	}
	log.Println("Цикл бэкапов завершен")
}


// waitForShutdown ожидает сигнал остановки (Ctrl+C, SIGTERM)
func waitForShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	sig := <-sigChan
	log.Printf("Получен сигнал: %v", sig)
	
	// Отменяем контекст - все горутины получат сигнал остановки
	cancel()
}

func cleanupOldBackups(
	ctx context.Context,
	st storage.Storage,
	notifiers map[string]notifier.Notifier,
	retentionDays int,
) {
	deleted, err := storage.CleanupOldBackups(ctx, st, retentionDays)
	if err != nil {
		msg := "Ошибка очистки старых бэкапов: " + err.Error()
		log.Println(msg)
		notifier.SendAll(notifiers, ctx, msg)
		return
	}
	
	if deleted > 0 {
		msg := fmt.Sprintf("Удалено %d старых бэкапов (старше %d дней)", deleted, retentionDays)
		log.Println(msg)
		notifier.SendAll(notifiers, ctx, msg)
	}
}

func getCronStringForCleanupFunc() string {
	return "* * 1 * * *"
}
