// internal/app/app.go
package app

import (
    "context"
    "fmt"
    "log/slog"
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
    "gorm.io/gorm"
)

type App struct {
    config     *config.BackupConfig
    db         *gorm.DB
    storage    storage.Storage
    backuppers map[string]backup.Backupper
    notifiers  map[string]notifier.Notifier
    backupRepo *database.GormBackupLogRepository
    scheduler  *scheduler.Scheduler
    ctx        context.Context
    cancel     context.CancelFunc
}

func New() (*App, error) {
    slog.Info("инициализация приложения")

    // 1. Загрузка конфига
    cfg, err := config.NewBackupConfig()
    if err != nil {
        return nil, fmt.Errorf("загрузка конфига: %w", err)
    }
    slog.Info("конфиг загружен", "cron_enable", cfg.CronEnable)

    // 2. Инициализация БД
    db, err := database.InitDB("backup_service.db")
    if err != nil {
        return nil, fmt.Errorf("инициализация БД: %w", err)
    }
    slog.Info("БД инициализирована")

    // 3. Инициализация хранилища
    st, err := storage.NewStorage(cfg.StorageType)
    if err != nil {
        return nil, fmt.Errorf("создание хранилища: %w", err)
    }
    slog.Info("хранилище создано", "type", cfg.StorageType)

    // 4. Инициализация нотифаеров и бэкапперов
    notifiers := notifier.InitNotifiers()
    backuppers := backup.InitBackuppers()
    backupRepo := database.NewGormBackupLogRepository(db)

    // 5. Контекст
    ctx, cancel := context.WithCancel(context.Background())

    return &App{
        config:     cfg,
        db:         db,
        storage:    st,
        backuppers: backuppers,
        notifiers:  notifiers,
        backupRepo: backupRepo,
        scheduler:  scheduler.New(),
        ctx:        ctx,
        cancel:     cancel,
    }, nil
}

func (a *App) Close() error {
    slog.Info("закрытие ресурсов")
    if a.db != nil {
        sqlDB, err := a.db.DB()
        if err != nil {
            return err
        }
        return sqlDB.Close()
    }
    return nil
}

func (a *App) Start() error {
    slog.Info("запуск сервиса бэкапов")

    // Проверка CronEnable
    if !a.config.CronEnable {
        return fmt.Errorf("планировщик отключен (CRON_ENABLE=false), используйте команду run для однократного запуска")
    }

    // Стартовый бэкап
    go func() {
        slog.Info("стартовый бэкап (проверка)")
        ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
        defer cancel()
        a.runBackupCycle(ctx)

        slog.Info("стартовая очистка (проверка)")
        cleanupCtx, cleanupCancel := context.WithTimeout(a.ctx, 1*time.Minute)
        defer cleanupCancel()
        a.cleanupOldBackups(cleanupCtx)
    }()
    time.Sleep(1 * time.Second)

    // Добавление задач в планировщик
    backupJob := func() {
        ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
        defer cancel()
        a.runBackupCycle(ctx)
    }

    if _, err := a.scheduler.AddJob(a.config.BackupSchedule, backupJob); err != nil {
        return fmt.Errorf("добавление backupJob: %w", err)
    }

    cleanupJob := func() {
        ctx, cancel := context.WithTimeout(a.ctx, 1*time.Minute)
        defer cancel()
        a.cleanupOldBackups(ctx)
    }

    if _, err := a.scheduler.AddJob(getCronStringForCleanup(), cleanupJob); err != nil {
        return fmt.Errorf("добавление cleanupJob: %w", err)
    }

    slog.Info("планировщик запущен", "backup_schedule", a.config.BackupSchedule)

    // Запуск планировщика
    a.scheduler.Start()
    defer a.scheduler.Stop(a.ctx)

    // Ожидание сигнала
    a.waitForShutdown()

    slog.Info("сервис остановлен")
    return nil
}

func (a *App) RunOnce() error {
    slog.Info("однократный запуск бэкапа")
    ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
    defer cancel()
    a.runBackupCycle(ctx)
    slog.Info("однократный запуск завершен")
    return nil
}

func (a *App) Cleanup() error {
    slog.Info("запуск очистки старых бэкапов")
    ctx, cancel := context.WithTimeout(a.ctx, 1*time.Minute)
    defer cancel()
    a.cleanupOldBackups(ctx)
    slog.Info("очистка завершена")
    return nil
}

func (a *App) Restore(backupName, backupType string) error {
    slog.Info("восстановление из бэкапа", "backup", backupName, "type", backupType)
       
    // Получаем бэкаппер по типу
    backupper, ok := a.backuppers[backupType]
    if !ok {
        return fmt.Errorf("не найден бэкаппер типа: %s", backupType)
    }
    
    // Скачиваем файл из хранилища
    localPath, err := a.storage.Download(a.ctx, backupName)
    if err != nil {
        return fmt.Errorf("получение бэкапа из хранилища: %w", err)
    }
    
    // Восстанавливаем
    restoredPath, err := backupper.RestoreBackup(a.ctx, localPath)
    if err != nil {
        return fmt.Errorf("восстановление: %w", err)
    }
    
    slog.Info("БД успешно восстановлена", "path", restoredPath)
    return nil
}

func (a *App) Stop() error {
    slog.Info("остановка сервиса (заглушка)")
    a.cancel()
    return nil
}

// --- Внутренние методы ---

func (a *App) runBackupCycle(ctx context.Context) {
	select {
	case <-ctx.Done():
		slog.Info("Бэкап пропущен: контекст отменен")
		return
	default:
	}

	slog.Info("Начало цикла бэкапов")

	for typ, backupper := range a.backuppers {
		// Проверяем контекст перед каждым бэкапом
		select {
		case <-ctx.Done():
			slog.Info("Бэкап %s прерван: %v", typ, ctx.Err())
			msg := "Бэкап " + typ + " прерван при остановке сервиса"
			notifier.SendAll(a.notifiers, ctx, msg)
			return
		default:
		}

		slog.Info(fmt.Sprintf("Запуск бэкапа: %v", typ))

		// Создание бэкапа
		localPath, err := backup.RunBackup(ctx, backupper, a.backupRepo, a.config.BackupPath, a.config.StorageType)
		if err != nil {
			msg := "Ошибка создания бэкапа " + typ + ": " + err.Error()
			notifier.SendAll(a.notifiers, ctx, msg)
			continue
		}

		// Сохранение в хранилище
		remotePath, err := a.storage.Save(ctx, localPath)
		if err != nil {
			msg := "Ошибка сохранения бэкапа " + typ + " в хранилище: " + err.Error()
			notifier.SendAll(a.notifiers, ctx, msg)
			continue
		}

		// Уведомление об успехе
		msg := "Бэкап " + typ + " сохранен: " + remotePath
		notifier.SendAll(a.notifiers, ctx, msg)

		slog.Info("Бэкап %s сохранен в хранилище: %s", typ, remotePath)
	}
	slog.Info("Цикл бэкапов завершен")
}

func (a *App) cleanupOldBackups(ctx context.Context) {
	deleted, err := storage.CleanupOldBackups(ctx, a.storage, a.config.RetentionDays)
	if err != nil {
		msg := "Ошибка очистки старых бэкапов: " + err.Error()
		slog.Info(msg)
		notifier.SendAll(a.notifiers, ctx, msg)
		return
	}
	
	if deleted > 0 {
		msg := fmt.Sprintf("Удалено %d старых бэкапов (старше %d дней)", deleted, a.config.RetentionDays)
		slog.Info(msg)
		notifier.SendAll(a.notifiers, ctx, msg)
	}
}

func (a *App) waitForShutdown() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    sig := <-sigChan
    slog.Info("получен сигнал", "signal", sig)
    a.cancel()
}

func getCronStringForCleanup() string {
    return "* * 1 * * *"
}