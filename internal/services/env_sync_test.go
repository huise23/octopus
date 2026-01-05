package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewEnvSyncService(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
	defer os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")

	service := NewEnvSyncService()

	if service == nil {
		t.Fatal("NewEnvSyncService() returned nil")
	}

	if service.client == nil {
		t.Error("NewEnvSyncService() did not create HTTP client")
	}

	if service.client.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", service.client.Timeout)
	}
}

func TestNewEnvSyncServiceWithEnv(t *testing.T) {
	// 设置环境变量
	testURL := "https://test-api.com"
	os.Setenv("OCTOPUS_ENV_SYNC_API_URL", testURL)
	defer os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")

	service := NewEnvSyncService()

	if service.apiURL != testURL {
		t.Errorf("Expected apiURL %s, got %s", testURL, service.apiURL)
	}
}

func TestSyncDomain(t *testing.T) {
	tests := []struct {
		name         string
		domain       string
		setEnvVar    bool
		mockResponse int
		expectError  bool
	}{
		{
			name:         "环境变量未设置",
			domain:       "api.example.com",
			setEnvVar:    false,
			mockResponse: 200,
			expectError:  false,
		},
		{
			name:         "成功同步",
			domain:       "api.example.com",
			setEnvVar:    true,
			mockResponse: 200,
			expectError:  false,
		},
		{
			name:         "API 返回错误状态码",
			domain:       "api.example.com",
			setEnvVar:    true,
			mockResponse: 500,
			expectError:  false, // 不会返回错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理并设置环境变量
			os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
			if tt.setEnvVar {
				// 创建 mock 服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.mockResponse)
				}))
				defer server.Close()

				os.Setenv("OCTOPUS_ENV_SYNC_API_URL", server.URL)
			}
			defer os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")

			// 创建服务
			service := NewEnvSyncService()

			// 执行测试
			err := service.SyncDomain(tt.domain)

			// 验证结果
			if (err != nil) != tt.expectError {
				t.Errorf("SyncDomain() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestSyncDomainAsync(t *testing.T) {
	// 创建 mock 服务器
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 设置环境变量
	os.Setenv("OCTOPUS_ENV_SYNC_API_URL", server.URL)
	defer os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")

	// 创建服务
	service := NewEnvSyncService()

	// 异步调用
	service.SyncDomainAsync("api.example.com")

	// 等待异步执行完成
	time.Sleep(100 * time.Millisecond)

	// 验证
	if callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", callCount)
	}
}

func TestSyncDomainAsyncMultiple(t *testing.T) {
	// 创建 mock 服务器
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 设置环境变量
	os.Setenv("OCTOPUS_ENV_SYNC_API_URL", server.URL)
	defer os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")

	// 创建服务
	service := NewEnvSyncService()

	// 异步调用多个域名
	domains := []string{
		"api1.example.com",
		"api2.example.com",
		"api3.example.com",
	}

	for _, domain := range domains {
		service.SyncDomainAsync(domain)
	}

	// 等待异步执行完成
	time.Sleep(200 * time.Millisecond)

	// 验证
	if callCount != len(domains) {
		t.Errorf("Expected %d API calls, got %d", len(domains), callCount)
	}
}
