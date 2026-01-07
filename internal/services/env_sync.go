package services

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bestruirui/octopus/internal/utils"
	"github.com/bestruirui/octopus/internal/utils/log"
)

const (
	envSyncAPIURLKey    = "OCTOPUS_ENV_SYNC_API_URL"
	envSyncProxyRuleKey = "OCTOPUS_ENV_SYNC_PROXY_RULE"  // 使用代理时的策略规则
	envSyncDirectRuleKey = "OCTOPUS_ENV_SYNC_DIRECT_RULE" // 不使用代理时的策略规则
	requestTimeout      = 5 * time.Second // 超时时间写死 5 秒

	// 默认策略规则：用户要求都默认为DIRECT
	defaultProxyRule  = "DIRECT"
	defaultDirectRule = "DIRECT"
)

// EnvSyncService 环境变量同步服务
type EnvSyncService struct {
	client     *http.Client
	apiURL     string
	proxyRule  string // 使用代理时的策略规则
	directRule string // 不使用代理时的策略规则
}

// NewEnvSyncService 创建环境同步服务实例
func NewEnvSyncService() *EnvSyncService {
	// 读取代理策略规则，如果未设置则使用默认值
	proxyRule := os.Getenv(envSyncProxyRuleKey)
	if proxyRule == "" {
		proxyRule = defaultProxyRule
	}

	directRule := os.Getenv(envSyncDirectRuleKey)
	if directRule == "" {
		directRule = defaultDirectRule
	}

	return &EnvSyncService{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		apiURL:     os.Getenv(envSyncAPIURLKey),
		proxyRule:  proxyRule,
		directRule: directRule,
	}
}

// SyncDomain 同步域名到环境变量
// useProxy: true表示使用代理，false表示不使用代理
// 如果环境变量未设置，则直接返回成功
func (s *EnvSyncService) SyncDomain(domain string, useProxy bool) error {
	// 检查环境变量是否设置
	if s.apiURL == "" {
		log.Debugf("Environment variable %s is not set, skipping sync", envSyncAPIURLKey)
		return nil
	}

	// 提取根域名
	rootDomain, err := utils.ExtractRootDomain(domain)
	if err != nil {
		log.Errorf("Failed to extract root domain from %s: %v", domain, err)
		return fmt.Errorf("failed to extract root domain: %w", err)
	}

	if rootDomain == "" {
		log.Warnf("Extracted root domain is empty, skipping sync")
		return nil
	}

	// 根据代理状态选择策略规则
	var rule string
	if useProxy {
		rule = s.proxyRule
	} else {
		rule = s.directRule
	}

	// 构建请求体
	payload := fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", rootDomain, rule)

	// 创建 HTTP 请求
	req, err := http.NewRequest("PUT", s.apiURL, bytes.NewBufferString(payload))
	if err != nil {
		log.Errorf("Failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	// 发送请求
	log.Infof("Syncing domain %s (root: %s) to %s with rule: %s", domain, rootDomain, s.apiURL, rule)
	resp, err := s.client.Do(req)
	if err != nil {
		log.Errorf("Failed to sync domain %s: %v", rootDomain, err)
		// 返回 nil 而不是 error，因为不希望影响主流程
		return nil
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Infof("Successfully synced domain %s (status: %d, rule: %s)", rootDomain, resp.StatusCode, rule)
	} else {
		log.Warnf("EnvSync API returned non-success status %d for domain %s", resp.StatusCode, rootDomain)
	}

	return nil
}

// SyncDomainAsync 异步同步域名到环境变量
// useProxy: true表示使用代理，false表示不使用代理
// 使用 goroutine 在后台执行，不会阻塞主流程
func (s *EnvSyncService) SyncDomainAsync(domain string, useProxy bool) {
	go func() {
		if err := s.SyncDomain(domain, useProxy); err != nil {
			log.Errorf("Async domain sync failed: %v", err)
		}
	}()
}
