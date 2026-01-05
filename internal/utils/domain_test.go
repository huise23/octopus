package utils

import (
	"testing"
)

func TestExtractRootDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "普通域名",
			input:   "example.com",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "子域名",
			input:   "api.example.com",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "多级子域名",
			input:   "v1.api.example.com",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "带 HTTPS 协议的 URL",
			input:   "https://api.example.com",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "带 HTTP 协议的 URL",
			input:   "http://api.example.com",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "带路径的 URL",
			input:   "https://api.example.com/path/to/resource",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "带端口号的 URL",
			input:   "https://api.example.com:8080",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "带路径和端口号的 URL",
			input:   "https://api.example.com:8080/path",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "复杂 URL",
			input:   "https://v1.api.example.com:8080/api/v1/users",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "co.uk 域名",
			input:   "api.co.uk",
			want:    "api.co.uk", // co.uk 是公共后缀，所以 api.co.uk 本身就是 eTLD+1
			wantErr: false,
		},
		{
			name:    "com.cn 域名",
			input:   "api.com.cn",
			want:    "api.com.cn", // 注意：Public Suffix List 可能将 com.cn 视为公共后缀
			wantErr: false,
		},
		{
			name:    "空字符串",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "带用户信息的 URL",
			input:   "https://user:pass@api.example.com",
			want:    "example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRootDomain(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRootDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRootDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		name string
		domain string
		want bool
	}{
		{
			name:   "有效域名",
			domain: "example.com",
			want:   true,
		},
		{
			name:   "有效子域名",
			domain: "api.example.com",
			want:   true,
		},
		{
			name:   "空字符串",
			domain: "",
			want:   false,
		},
		{
			name:   "无效域名",
			domain: "invalid..domain",
			want:   false,
		},
		{
			name:   "带协议的 URL（publicsuffix 会将其视为有效）",
			domain: "https://example.com",
			want:   true, // publicsuffix 库将此视为有效域名
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDomain(tt.domain); got != tt.want {
				t.Errorf("IsValidDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}
