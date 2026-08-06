package notifier

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"backup-service/internal/config"
	configMocks "backup-service/internal/config/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper позволяет гибко переопределять поведение http-транспорта.
type mockRoundTripper struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

func Test_newTelegramNotifier(t *testing.T) {
	// Не используем t.Parallel() на уровне всей функции, так как меняем глобальную переменную.
	// Каждый подтест изолирован своей подменой и cleanup.

	tests := []struct {
		name             string
		setupMock        func(m *configMocks.MockConfigLoader)
		wantErr          bool
		expectedErr      error
		checkNotifier    bool
		expectedBotToken string
		expectedChatID   string
	}{
		{
			name: "ошибка валидации: пустой токен",
			setupMock: func(m *configMocks.MockConfigLoader) {
				// TELEGRAM_ENABLE = true
				m.On("GetBool", []string{"telegram", "enable"}, false).Return(true)
				// TELEGRAM_BOT_TOKEN = "" (пустой)
				m.On("GetString", []string{"telegram", "bot_token"}, "").Return("")
				// TELEGRAM_CHAT_ID = "some_chat"
				m.On("GetString", []string{"telegram", "chat_id"}, "").Return("some_chat")
			},
			wantErr:     true,
			expectedErr: config.ErrEmptyTelegramBotToken,
		},
		{
			name: "телеграм выключен",
			setupMock: func(m *configMocks.MockConfigLoader) {
				m.On("GetBool", []string{"telegram", "enable"}, false).Return(false)
				// Остальные вызовы не должны происходить, но если произойдут – тест упадёт из-за неожиданных вызовов
			},
			wantErr:     true,
			expectedErr: ErrTelegramDisabled,
		},
		{
			name: "успешное создание",
			setupMock: func(m *configMocks.MockConfigLoader) {
				m.On("GetBool", []string{"telegram", "enable"}, false).Return(true)
				m.On("GetString", []string{"telegram", "bot_token"}, "").Return("test-token")
				m.On("GetString", []string{"telegram", "chat_id"}, "").Return("test-chat")
			},
			wantErr:          false,
			checkNotifier:    true,
			expectedBotToken: "test-token",
			expectedChatID:   "test-chat",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Готовим мок и подменяем глобальный лоадер
			mockLoader := configMocks.NewMockConfigLoader(t)
			tt.setupMock(mockLoader)

			telegramGetConfigLoader = func () config.ConfigLoader {
				return mockLoader
			}

			originalGetConfigLoader := telegramGetConfigLoader
			telegramGetConfigLoader = func () config.ConfigLoader {
				return mockLoader
			}
			t.Cleanup(func() { telegramGetConfigLoader = originalGetConfigLoader })

			notifier, err := newTelegramNotifier()

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, notifier)

				if tt.checkNotifier {
					tg, ok := notifier.(*TelegramNotifier)
					require.True(t, ok, "должен быть *TelegramNotifier")
					assert.Equal(t, tt.expectedBotToken, tg.botToken)
					assert.Equal(t, tt.expectedChatID, tg.chatID)
					assert.NotNil(t, tg.client)
					assert.Equal(t, 10*time.Second, tg.client.Timeout)
				}
			}
		})
	}
}

func TestTelegramNotifier_Send(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		botToken    string
		chatID      string
		ctx         context.Context
		message     string
		transport   http.RoundTripper
		wantErr     bool
		errContains string
	}{
		// --- Успешный сценарий ---
		{
			name:     "успешная отправка (HTTP 200)",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "Hello!",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr: false,
		},

		// --- Ошибки контекста ---
		{
			name:     "контекст уже отменён",
			botToken: "valid_token",
			chatID:   "123456",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			message: "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, nil
				},
			},
			wantErr:     true,
			errContains: context.Canceled.Error(),
		},
		{
			name:     "контекст с истекшим дедлайном",
			botToken: "valid_token",
			chatID:   "123456",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer cancel()
				return ctx
			}(),
			message: "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, nil
				},
			},
			wantErr:     true,
			errContains: context.DeadlineExceeded.Error(),
		},

		// --- Ошибка формирования запроса ---
		{
			name:     "невалидный URL (токен с переносом строки)",
			botToken: "токен\nс_переносом",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка создания запроса:",
		},

		// --- Сетевые ошибки при выполнении ---
		{
			name:     "сетевая ошибка (имитация обрыва соединения)",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("connection refused")
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки запроса:",
		},

		// --- HTTP-статусы, отличные от 200 (прокси / серверные ошибки) ---
		{
			name:     "HTTP 400 Bad Request",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки сообщения: статус 400",
		},
		{
			name:     "HTTP 401 Unauthorized",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки сообщения: статус 401",
		},
		{
			name:     "HTTP 403 Forbidden",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusForbidden,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки сообщения: статус 403",
		},
		{
			name:     "HTTP 404 Not Found",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки сообщения: статус 404",
		},
		{
			name:     "HTTP 429 Too Many Requests",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusTooManyRequests,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки сообщения: статус 429",
		},
		{
			name:     "HTTP 500 Internal Server Error",
			botToken: "valid_token",
			chatID:   "123456",
			ctx:      context.Background(),
			message:  "test",
			transport: &mockRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       http.NoBody,
					}, nil
				},
			},
			wantErr:     true,
			errContains: "Ошибка отправки сообщения: статус 500",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &http.Client{Transport: tt.transport}
			notifier := NewTelegramNotifier(tt.botToken, tt.chatID, client)
			err := notifier.Send(tt.ctx, tt.message)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.True(t, strings.Contains(err.Error(), tt.errContains),
						"ожидалась ошибка, содержащая %q; получена: %q", tt.errContains, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
