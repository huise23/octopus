package model

// SensitiveFilterRule 敏感信息过滤规则
type SensitiveFilterRule struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"not null"`
	Pattern     string `json:"pattern" gorm:"not null"`
	Replacement string `json:"replacement" gorm:"not null"`
	Enabled     bool   `json:"enabled" gorm:"default:true"`
	BuiltIn     bool   `json:"built_in" gorm:"default:false"`
	Priority    int    `json:"priority" gorm:"default:0"`
}

// DefaultSensitiveFilterRules 返回内置的默认过滤规则
func DefaultSensitiveFilterRules() []SensitiveFilterRule {
	return []SensitiveFilterRule{
		{Name: "OpenAI API Key", Pattern: `sk-[a-zA-Z0-9_-]{20,}`, Replacement: "[FILTERED:API_KEY]", Enabled: true, BuiltIn: true, Priority: 100},
		{Name: "Anthropic API Key", Pattern: `sk-ant-[a-zA-Z0-9_-]{20,}`, Replacement: "[FILTERED:API_KEY]", Enabled: true, BuiltIn: true, Priority: 100},
		{Name: "Database URL", Pattern: `(mysql|postgres|postgresql|mongodb|redis)://[^\s"'<>]+`, Replacement: "[FILTERED:DB_URL]", Enabled: true, BuiltIn: true, Priority: 90},
		{Name: "Bearer JWT Token", Pattern: `Bearer\s+[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`, Replacement: "[FILTERED:TOKEN]", Enabled: true, BuiltIn: true, Priority: 80},
		{Name: "AWS Access Key", Pattern: `AKIA[0-9A-Z]{16}`, Replacement: "[FILTERED:AWS_KEY]", Enabled: true, BuiltIn: true, Priority: 70},
		{Name: "GitHub Token", Pattern: `ghp_[a-zA-Z0-9]{36}`, Replacement: "[FILTERED:GH_TOKEN]", Enabled: true, BuiltIn: true, Priority: 70},
		{Name: "Private Key Header", Pattern: `-----BEGIN[A-Z ]*PRIVATE KEY-----`, Replacement: "[FILTERED:PRIVATE_KEY]", Enabled: true, BuiltIn: true, Priority: 60},
		{Name: "Password JSON Field", Pattern: `"password"\s*:\s*"[^"]*"`, Replacement: `"password":"[FILTERED]"`, Enabled: false, BuiltIn: true, Priority: 50},
	}
}
