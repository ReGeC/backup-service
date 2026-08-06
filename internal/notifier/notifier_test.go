package notifier_test

import (
	"backup-service/internal/notifier"
	mocks "backup-service/internal/notifier/mocks"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRegistry(t *testing.T) {
	t.Helper()

	notifier.ResetRegistry()

	t.Cleanup(func() {
		notifier.ResetRegistry()
	})
}

func TestRegister(t *testing.T) {
	resetRegistry(t)

	t.Run("register new notifier type", func(t *testing.T) {
		factory := func() (notifier.Notifier, error) {
			return &mocks.MockNotifier{}, nil
		}

		notifier.Register("test", factory)

		n, err := notifier.NewNotifier("test")
		require.NoError(t, err)
		assert.NotNil(t, n)
	})

	t.Run("register multiple notifier types", func(t *testing.T) {
		factory1 := func() (notifier.Notifier, error) {
			return &mocks.MockNotifier{}, nil
		}
		factory2 := func() (notifier.Notifier, error) {
			return &mocks.MockNotifier{}, nil
		}

		notifier.Register("type1", factory1)
		notifier.Register("type2", factory2)

		n1, err := notifier.NewNotifier("type1")
		require.NoError(t, err)
		assert.NotNil(t, n1)

		n2, err := notifier.NewNotifier("type2")
		require.NoError(t, err)
		assert.NotNil(t, n2)
	})

	t.Run("register overwrites existing type", func(t *testing.T) {
		counter := 0
		factoryOld := func() (notifier.Notifier, error) {
			counter = 1
			return &mocks.MockNotifier{}, nil
		}
		factoryNew := func() (notifier.Notifier, error) {
			counter = 2
			return &mocks.MockNotifier{}, nil
		}

		notifier.Register("same", factoryOld)
		notifier.Register("same", factoryNew)

		n, err := notifier.NewNotifier("same")
		require.NoError(t, err)
		assert.NotNil(t, n)
		assert.Equal(t, 2, counter)
	})
}

func TestResetRegistry(t *testing.T) {
	resetRegistry(t)

	t.Run("reset clears registry", func(t *testing.T) {
		factory := func() (notifier.Notifier, error) {
			return &mocks.MockNotifier{}, nil
		}
		notifier.Register("test", factory)

		_, err := notifier.NewNotifier("test")
		require.NoError(t, err)

		notifier.ResetRegistry()

		_, err = notifier.NewNotifier("test")
		require.Error(t, err)
		assert.EqualError(t, err, "Неизвестный тип уведомителя: test")
	})

	t.Run("reset empty registry", func(t *testing.T) {
		notifier.ResetRegistry()

		_, err := notifier.NewNotifier("any")
		require.Error(t, err)
		assert.EqualError(t, err, "Неизвестный тип уведомителя: any")
	})
}

func TestNewNotifier(t *testing.T) {
	resetRegistry(t)

	t.Run("create notifier for registered type", func(t *testing.T) {
		mockNotifier := &mocks.MockNotifier{}
		factory := func() (notifier.Notifier, error) {
			return mockNotifier, nil
		}

		notifier.Register("telegram", factory)

		n, err := notifier.NewNotifier("telegram")
		require.NoError(t, err)
		assert.Equal(t, mockNotifier, n)
	})

	t.Run("create notifier with factory returning error", func(t *testing.T) {
		expectedErr := assert.AnError
		factory := func() (notifier.Notifier, error) {
			return nil, expectedErr
		}

		notifier.Register("failing", factory)

		n, err := notifier.NewNotifier("failing")
		require.Error(t, err)
		assert.Nil(t, n)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("create notifier for unknown type returns error", func(t *testing.T) {
		n, err := notifier.NewNotifier("unknown")
		require.Error(t, err)
		assert.Nil(t, n)
		assert.EqualError(t, err, "Неизвестный тип уведомителя: unknown")
	})
}

func TestInitNotifiers(t *testing.T) {
	t.Run("init all notifiers successfully", func(t *testing.T) {
		resetRegistry(t)
		mock1 := &mocks.MockNotifier{}
		mock2 := &mocks.MockNotifier{}

		notifier.Register("type1", func() (notifier.Notifier, error) {
			return mock1, nil
		})
		notifier.Register("type2", func() (notifier.Notifier, error) {
			return mock2, nil
		})

		notifiers := notifier.InitNotifiers()

		assert.Len(t, notifiers, 2)
		assert.Equal(t, mock1, notifiers["type1"])
		assert.Equal(t, mock2, notifiers["type2"])
	})

	t.Run("init with disabled notifier", func(t *testing.T) {
		resetRegistry(t)
		notifier.Register("enabled", func() (notifier.Notifier, error) {
			return &mocks.MockNotifier{}, nil
		})
		notifier.Register("disabled", func() (notifier.Notifier, error) {
			return nil, notifier.ErrDisabled
		})

		notifiers := notifier.InitNotifiers()

		assert.Len(t, notifiers, 1)
		assert.Contains(t, notifiers, "enabled")
		assert.NotContains(t, notifiers, "disabled")
	})

	t.Run("init with failing notifier", func(t *testing.T) {
		resetRegistry(t)
		notifier.Register("good", func() (notifier.Notifier, error) {
			return &mocks.MockNotifier{}, nil
		})
		notifier.Register("bad", func() (notifier.Notifier, error) {
			return nil, assert.AnError
		})

		notifiers := notifier.InitNotifiers()

		assert.Len(t, notifiers, 1)
		assert.Contains(t, notifiers, "good")
		assert.NotContains(t, notifiers, "bad")
	})

	t.Run("init with empty registry", func(t *testing.T) {
		resetRegistry(t)
		notifiers := notifier.InitNotifiers()
		assert.Empty(t, notifiers)
	})
}

func TestSendAll(t *testing.T) {
	ctx := context.Background()
	message := "test message"

	t.Run("send to all notifiers successfully", func(t *testing.T) {
		mock1 := mocks.NewMockNotifier(t)
		mock2 := mocks.NewMockNotifier(t)

		mock1.On("Send", ctx, message).Return(nil)
		mock2.On("Send", ctx, message).Return(nil)

		notifiers := map[string]notifier.Notifier{
			"type1": mock1,
			"type2": mock2,
		}

		notifier.SendAll(notifiers, ctx, message)

		mock1.AssertExpectations(t)
		mock2.AssertExpectations(t)
	})

	t.Run("send with one notifier failing", func(t *testing.T) {
		mock1 := mocks.NewMockNotifier(t)
		mock2 := mocks.NewMockNotifier(t)

		expectedErr := assert.AnError
		mock1.On("Send", ctx, message).Return(nil)
		mock2.On("Send", ctx, message).Return(expectedErr)

		notifiers := map[string]notifier.Notifier{
			"type1": mock1,
			"type2": mock2,
		}

		// Функция не должна паниковать или возвращать ошибку
		notifier.SendAll(notifiers, ctx, message)

		mock1.AssertExpectations(t)
		mock2.AssertExpectations(t)
	})

	t.Run("send with all notifiers failing", func(t *testing.T) {
		mock1 := mocks.NewMockNotifier(t)
		mock2 := mocks.NewMockNotifier(t)

		mock1.On("Send", ctx, message).Return(assert.AnError)
		mock2.On("Send", ctx, message).Return(assert.AnError)

		notifiers := map[string]notifier.Notifier{
			"type1": mock1,
			"type2": mock2,
		}

		// Функция не должна паниковать
		notifier.SendAll(notifiers, ctx, message)

		mock1.AssertExpectations(t)
		mock2.AssertExpectations(t)
	})

	t.Run("send with empty notifiers map", func(t *testing.T) {
		notifiers := map[string]notifier.Notifier{}

		// Не должно паниковать
		notifier.SendAll(notifiers, ctx, message)
	})

	t.Run("send with nil notifiers map", func(t *testing.T) {
		// Не должно паниковать
		notifier.SendAll(nil, ctx, message)
	})

	t.Run("send with context cancellation", func(t *testing.T) {
		mock := mocks.NewMockNotifier(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Отменяем контекст

		mock.On("Send", ctx, message).Return(context.Canceled)

		notifiers := map[string]notifier.Notifier{
			"type1": mock,
		}

		notifier.SendAll(notifiers, ctx, message)

		mock.AssertExpectations(t)
	})
}

func TestNotifierInterface(t *testing.T) {
	t.Run("mock notifier implements interface", func(t *testing.T) {
		ctx := context.Background()
		mockNotifier := mocks.NewMockNotifier(t)

		message := "test notification"
		mockNotifier.On("Send", ctx, message).Return(nil)

		err := mockNotifier.Send(ctx, message)
		assert.NoError(t, err)

		mockNotifier.AssertExpectations(t)
	})

	t.Run("mock notifier with error", func(t *testing.T) {
		ctx := context.Background()
		mockNotifier := mocks.NewMockNotifier(t)

		expectedErr := assert.AnError
		message := "test notification"
		mockNotifier.On("Send", ctx, message).Return(expectedErr)

		err := mockNotifier.Send(ctx, message)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		mockNotifier.AssertExpectations(t)
	})
}

func TestErrDisabled(t *testing.T) {
	t.Run("ErrDisabled error", func(t *testing.T) {
		err := notifier.ErrDisabled
		assert.Error(t, err)
		assert.Equal(t, "Notifier is disabled: ", err.Error())
	})

	t.Run("check if ErrDisabled is returned", func(t *testing.T) {
		resetRegistry(t)

		notifier.Register("disabled", func() (notifier.Notifier, error) {
			return nil, notifier.ErrDisabled
		})

		_, err := notifier.NewNotifier("disabled")
		assert.Error(t, err)
		assert.True(t, err == notifier.ErrDisabled)
	})
}
