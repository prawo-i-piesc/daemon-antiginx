package worker

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockScanListener struct {
	mock.Mock
}

func (m *MockScanListener) Listen(workerId int) {
	m.Called(workerId)
}

var configureWorkersTests = []struct {
	name        string
	envValue    string
	setupMock   func(m *MockScanListener)
	expectError bool
}{
	{
		name:     "Happy path, custom worker pool size",
		envValue: "5",
		setupMock: func(m *MockScanListener) {
			for i := 0; i < 5; i++ {
				m.On("Listen", i).Once()
			}
		},
		expectError: false,
	},
	{
		name:     "Happy path, default worker pool size fallback",
		envValue: "",
		setupMock: func(m *MockScanListener) {
			for i := 0; i < 10; i++ {
				m.On("Listen", i).Once()
			}
		},
		expectError: false,
	},
	{
		name:     "Should return error on invalid env value",
		envValue: "not-a-number",
		setupMock: func(m *MockScanListener) {
		},
		expectError: true,
	},
}

func TestConfigureWorkers(t *testing.T) {
	for _, tt := range configureWorkersTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(workerPoolEnvName, tt.envValue)

			mockListener := new(MockScanListener)
			tt.setupMock(mockListener)

			wg, err := ConfigureWorkers(mockListener)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, wg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, wg)

				wg.Wait()
			}

			mockListener.AssertExpectations(t)
		})
	}
}
