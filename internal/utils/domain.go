package utils

import (
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// ExtractRootDomain 从完整的 URL 或域名中提取根域名
// 例如：
//   - "https://api.example.com/path" -> "example.com"
//   - "api.example.com" -> "example.com"
//   - "example.com" -> "example.com"
//   - "https://api.co.uk" -> "api.co.uk" (注意：co.uk 是公共后缀)
func ExtractRootDomain(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// 如果包含协议，使用 net/url 解析
	if strings.Contains(input, "://") {
		parsedURL, err := url.Parse(input)
		if err != nil {
			return "", err
		}
		input = parsedURL.Host
	} else {
		// 移除路径部分（如果没有协议）
		if idx := strings.Index(input, "/"); idx != -1 {
			input = input[:idx]
		}
	}

	// 移除端口号
	if idx := strings.Index(input, ":"); idx != -1 {
		input = input[:idx]
	}

	// 如果包含用户信息（如 user:pass@host），移除它
	if idx := strings.Index(input, "@"); idx != -1 {
		input = input[idx+1:]
	}

	// 使用 Public Suffix List 提取根域名
	rootDomain, err := publicsuffix.EffectiveTLDPlusOne(input)
	if err != nil {
		return "", err
	}

	return rootDomain, nil
}

// IsValidDomain 检查域名是否有效
func IsValidDomain(domain string) bool {
	if domain == "" {
		return false
	}

	_, err := publicsuffix.EffectiveTLDPlusOne(domain)
	return err == nil
}
