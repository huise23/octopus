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
	os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
	os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")
	defer func() {
		os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
		os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
		os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")
	}()

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

	// 验证默认策略规则
	if service.proxyRule != "DIRECT" {
		t.Errorf("Expected proxyRule DIRECT, got %s", service.proxyRule)
	}

	if service.directRule != "DIRECT" {
		t.Errorf("Expected directRule DIRECT, got %s", service.directRule)
	}
}

func TestNewEnvSyncServiceWithEnv(t *testing.T) {
	// 设置环境变量
	testURL := "https://test-api.com"
	testProxyRule := "MyProxy"
	testDirectRule := "MyDirect"

	os.Setenv("OCTOPUS_ENV_SYNC_API_URL", testURL)
	os.Setenv("OCTOPUS_ENV_SYNC_PROXY_RULE", testProxyRule)
	os.Setenv("OCTOPUS_ENV_SYNC_DIRECT_RULE", testDirectRule)
	defer func() {
		os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
		os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
		os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")
	}()

	service := NewEnvSyncService()

	if service.apiURL != testURL {
		t.Errorf("Expected apiURL %s, got %s", testURL, service.apiURL)
	}

	if service.proxyRule != testProxyRule {
		t.Errorf("Expected proxyRule %s, got %s", testProxyRule, service.proxyRule)
	}

	if service.directRule != testDirectRule {
		t.Errorf("Expected directRule %s, got %s", testDirectRule, service.directRule)
	}
}

func TestSyncDomain(t *testing.T) {
	tests := []struct {
		name         string
		domain       string
		useProxy     bool
		setEnvVar    bool
		proxyRule    string
		directRule   string
		mockResponse int
		expectError  bool
		expectPayload string
	}{
		{
			name:         "环境变量未设置",
			domain:       "api.example.com",
			useProxy:     false,
			setEnvVar:    false,
			proxyRule:    "",
			directRule:   "",
			mockResponse: 200,
			expectError:  false,
			expectPayload: "",
		},
		{
			name:         "使用代理-默认规则",
			domain:       "api.example.com",
			useProxy:     true,
			setEnvVar:    true,
			proxyRule:    "", // 使用默认值DIRECT
			directRule:   "",
			mockResponse: 200,
			expectError:  false,
			expectPayload: "DOMAIN-SUFFIX,example.com,DIRECT",
		},
		{
			name:         "不使用代理-默认规则",
			domain:       "api.example.com",
			useProxy:     false,
			setEnvVar:    true,
			proxyRule:    "",
			directRule:   "", // 使用默认值DIRECT
			mockResponse: 200,
			expectError:  false,
			expectPayload: "DOMAIN-SUFFIX,example.com,DIRECT",
		},
		{
			name:         "使用代理-自定义规则",
			domain:       "api.example.com",
			useProxy:     true,
			setEnvVar:    true,
			proxyRule:    "MyProxy",
			directRule:   "MyDirect",
			mockResponse: 200,
			expectError:  false,
			expectPayload: "DOMAIN-SUFFIX,example.com,MyProxy",
		},
		{
			name:         "不使用代理-自定义规则",
			domain:       "api.example.com",
			useProxy:     false,
			setEnvVar:    true,
			proxyRule:    "MyProxy",
			directRule:   "MyDirect",
			mockResponse: 200,
			expectError:  false,
			expectPayload: "DOMAIN-SUFFIX,example.com,MyDirect",
		},
		{
			name:         "API 返回错误状态码",
			domain:       "api.example.com",
			useProxy:     false,
			setEnvVar:    true,
			proxyRule:    "",
			directRule:   "",
			mockResponse: 500,
			expectError:  false, // 不会返回错误
			expectPayload: "DOMAIN-SUFFIX,example.com,DIRECT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理并设置环境变量
			os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
			os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
			os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")

			var receivedPayload string
			if tt.setEnvVar {
				// 创建 mock 服务器
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 读取请求体以验证payload
					body := make([]byte, r.ContentLength)
					r.Body.Read(body)
					receivedPayload = string(body)
					w.WriteHeader(tt.mockResponse)
				}))
				defer server.Close()

				os.Setenv("OCTOPUS_ENV_SYNC_API_URL", server.URL)
				if tt.proxyRule != "" {
					os.Setenv("OCTOPUS_ENV_SYNC_PROXY_RULE", tt.proxyRule)
				}
				if tt.directRule != "" {
					os.Setenv("OCTOPUS_ENV_SYNC_DIRECT_RULE", tt.directRule)
				}
			}
			defer func() {
				os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
				os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
				os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")
			}()

			// 创建服务
			service := NewEnvSyncService()

			// 执行测试
			err := service.SyncDomain(tt.domain, tt.useProxy)

			// 验证结果
			if (err != nil) != tt.expectError {
				t.Errorf("SyncDomain() error = %v, expectError %v", err, tt.expectError)
			}

			// 验证payload（如果设置了环境变量）
			if tt.setEnvVar && tt.expectPayload != "" {
				if receivedPayload != tt.expectPayload {
					t.Errorf("Expected payload %s, got %s", tt.expectPayload, receivedPayload)
				}
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
	service.SyncDomainAsync("api.example.com", false)

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

	// 异步调用多个域名，混合代理状态
	domains := []struct {
		domain   string
		useProxy bool
	}{
		{"api1.example.com", false},
		{"api2.example.com", true},
		{"api3.example.com", false},
	}

	for _, d := range domains {
		service.SyncDomainAsync(d.domain, d.useProxy)
	}

	// 等待异步执行完成
	time.Sleep(200 * time.Millisecond)

	// 验证
	if callCount != len(domains) {
		t.Errorf("Expected %d API calls, got %d", len(domains), callCount)
	}
}

// TestProxyRuleSelection 测试策略规则的选择逻辑
func TestProxyRuleSelection(t *testing.T) {
	tests := []struct {
		name          string
		proxyRule     string
		directRule    string
		useProxy      bool
		expectedRule  string
	}{
		{
			name:         "使用代理时选择代理规则",
			proxyRule:    "MyProxy",
			directRule:   "MyDirect",
			useProxy:     true,
			expectedRule: "MyProxy",
		},
		{
			name:         "不使用代理时选择直连规则",
			proxyRule:    "MyProxy",
			directRule:   "MyDirect",
			useProxy:     false,
			expectedRule: "MyDirect",
		},
		{
			name:         "默认值测试-使用代理",
			proxyRule:    "",
			directRule:   "",
			useProxy:     true,
			expectedRule: "DIRECT", // 默认值
		},
		{
			name:         "默认值测试-不使用代理",
			proxyRule:    "",
			directRule:   "",
			useProxy:     false,
			expectedRule: "DIRECT", // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPayload string

			// 创建 mock 服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := make([]byte, r.ContentLength)
				r.Body.Read(body)
				receivedPayload = string(body)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// 清理并设置环境变量
			os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
			os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
			os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")

			os.Setenv("OCTOPUS_ENV_SYNC_API_URL", server.URL)
			if tt.proxyRule != "" {
				os.Setenv("OCTOPUS_ENV_SYNC_PROXY_RULE", tt.proxyRule)
			}
			if tt.directRule != "" {
				os.Setenv("OCTOPUS_ENV_SYNC_DIRECT_RULE", tt.directRule)
			}

			defer func() {
				os.Unsetenv("OCTOPUS_ENV_SYNC_API_URL")
				os.Unsetenv("OCTOPUS_ENV_SYNC_PROXY_RULE")
				os.Unsetenv("OCTOPUS_ENV_SYNC_DIRECT_RULE")
			}()

			// 创建服务并调用
			service := NewEnvSyncService()
			err := service.SyncDomain("api.example.com", tt.useProxy)

			// 验证结果
			if err != nil {
				t.Errorf("SyncDomain() error = %v", err)
			}

			expectedPayload := "DOMAIN-SUFFIX,example.com," + tt.expectedRule
			if receivedPayload != expectedPayload {
				t.Errorf("Expected payload %s, got %s", expectedPayload, receivedPayload)
			}
		})
	}
}
