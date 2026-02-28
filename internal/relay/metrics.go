package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// RelayMetrics 统一管理请求的日志记录和统计信息
type RelayMetrics struct {
	// 基础信息
	ChannelID      int
	APIKeyID       int
	ChannelName    string // 渠道名称
	RequestModel   string // 请求的模型名称
	ActualModel    string // 实际使用的模型名称
	StartTime      time.Time
	FirstTokenTime time.Time // 首个 Token 时间（流式场景）

	// 请求和响应内容
	InternalRequest  *transformerModel.InternalLLMRequest
	InternalResponse *transformerModel.InternalLLMResponse

	// 统计指标
	Stats model.StatsMetrics

	// 重试信息
	Attempts []model.ChannelAttempt

	// 探测重试的 token/费用统计
	Probe ProbeStats
}
type ProbeStats struct {
	InputTokens  int
	OutputTokens int
	Cost         float64
}

// NewRelayMetrics 创建新的 RelayMetrics
func NewRelayMetrics(requestModel string) *RelayMetrics {
	return &RelayMetrics{
		RequestModel: requestModel,
		StartTime:    time.Now(),
	}
}

func (m *RelayMetrics) SetAPIKeyID(apiKeyID int) {
	m.APIKeyID = apiKeyID
}

// SetChannel 设置通道信息
func (m *RelayMetrics) SetChannel(channelID int, channelName string, actualModel string) {
	m.ChannelID = channelID
	m.ChannelName = channelName
	m.ActualModel = actualModel
}

// SetFirstTokenTime 设置首个 Token 时间
func (m *RelayMetrics) SetFirstTokenTime(t time.Time) {
	m.FirstTokenTime = t
}

// SetInternalRequest 设置内部请求
func (m *RelayMetrics) SetInternalRequest(req *transformerModel.InternalLLMRequest) {
	m.InternalRequest = req
}

// AddAttempt 记录单次渠道尝试的信息
func (m *RelayMetrics) AddAttempt(round int, attemptNum int, success bool, err error, duration time.Duration) {
	attempt := model.ChannelAttempt{
		ChannelID:   m.ChannelID,
		ChannelName: m.ChannelName,
		ModelName:   m.ActualModel,
		Round:       round,
		AttemptNum:  attemptNum,
		Success:     success,
		Duration:    int(duration.Milliseconds()),
	}
	if err != nil {
		attempt.Error = err.Error()
	}
	m.Attempts = append(m.Attempts, attempt)
	m.saveStats(success, duration)
}

// SetInternalResponse 设置内部响应并计算费用
func (m *RelayMetrics) SetInternalResponse(resp *transformerModel.InternalLLMResponse) {
	m.InternalResponse = resp

	// 从响应中提取 Usage 并计算费用
	if resp == nil || resp.Usage == nil {
		return
	}

	usage := resp.Usage
	m.Stats.InputToken = usage.PromptTokens
	m.Stats.OutputToken = usage.CompletionTokens

	// 计算费用
	modelPrice := price.GetLLMPrice(m.ActualModel)
	if modelPrice == nil {
		return
	}
	if usage.PromptTokensDetails == nil {
		usage.PromptTokensDetails = &transformerModel.PromptTokensDetails{
			CachedTokens: 0,
		}
	}
	if usage.AnthropicUsage {
		m.Stats.InputCost = (float64(usage.PromptTokensDetails.CachedTokens)*modelPrice.CacheRead +
			float64(usage.PromptTokens)*modelPrice.Input +
			float64(usage.CacheCreationInputTokens)*modelPrice.CacheWrite) * 1e-6
	} else {
		m.Stats.InputCost = (float64(usage.PromptTokensDetails.CachedTokens)*modelPrice.CacheRead + float64(usage.PromptTokens-usage.PromptTokensDetails.CachedTokens)*modelPrice.Input) * 1e-6
	}
	m.Stats.OutputCost = float64(usage.CompletionTokens) * modelPrice.Output * 1e-6
}

// Save 保存日志和统计信息
// success: 请求是否成功
// err: 失败时的错误信息，成功时为 nil
// successfulRound: 成功的轮次 (1-3)，失败时为 0
func (m *RelayMetrics) Save(ctx context.Context, success bool, err error, successfulRound int) {
	duration := time.Since(m.StartTime)

	// 保存日志
	m.saveLog(ctx, err, duration, successfulRound)
}

// saveStats 保存统计信息
func (m *RelayMetrics) saveStats(success bool, duration time.Duration) {
	if success {
		m.Stats.RequestSuccess = 1
	} else {
		m.Stats.RequestFailed = 1
	}
	m.Stats.ProbeInputToken += int64(m.Probe.InputTokens)
	m.Stats.ProbeOutputToken += int64(m.Probe.OutputTokens)
	m.Stats.ProbeCost += m.Probe.Cost
	m.Stats.WaitTime = duration.Milliseconds()

	channelID, channelName := finalChannel(m.Attempts, m.ChannelID, m.ChannelName)
	op.StatsTotalUpdate(m.Stats)
	op.StatsHourlyUpdate(m.Stats)
	op.StatsDailyUpdate(context.Background(), m.Stats)
	op.StatsAPIKeyUpdate(m.APIKeyID, m.Stats)
	op.StatsChannelUpdate(channelID, m.Stats)

	log.Infof("relay complete: model=%s, channel=%d(%s), success=%t, duration=%dms, input_token=%d, output_token=%d, input_cost=%f, output_cost=%f, total_cost=%f, probe_token=%d, probe_cost=%f, attempts=%d",
		m.RequestModel, channelID, channelName, success, duration.Milliseconds(),
		m.Stats.InputToken, m.Stats.OutputToken,
		m.Stats.InputCost, m.Stats.OutputCost, m.Stats.InputCost+m.Stats.OutputCost,
		m.Stats.ProbeInputToken+m.Stats.ProbeOutputToken, m.Stats.ProbeCost, len(m.Attempts))

}

// saveLog 保存日志
func (m *RelayMetrics) saveLog(ctx context.Context, err error, duration time.Duration, successfulRound int) {
	relayLog := model.RelayLog{
		Time:             m.StartTime.Unix(),
		RequestModelName: m.RequestModel,
		ChannelName:      m.ChannelName,
		ChannelId:        m.ChannelID,
		ActualModelName:  m.ActualModel,
		UseTime:          int(duration.Milliseconds()),
		Attempts:         m.Attempts,
		TotalAttempts:    len(m.Attempts),
		SuccessfulRound:  successfulRound,
	}

	// 设置首字时间（流式场景）
	if !m.FirstTokenTime.IsZero() {
		relayLog.Ftut = int(m.FirstTokenTime.Sub(m.StartTime).Milliseconds())
	}

	// 设置 Usage 信息
	if m.InternalResponse != nil && m.InternalResponse.Usage != nil {
		relayLog.InputTokens = int(m.InternalResponse.Usage.PromptTokens)
		relayLog.OutputTokens = int(m.InternalResponse.Usage.CompletionTokens)
		relayLog.Cost = m.Stats.InputCost + m.Stats.OutputCost
	}

	// 设置请求内容
	if m.InternalRequest != nil {
		if reqJSON, jsonErr := json.Marshal(m.InternalRequest); jsonErr == nil {
			relayLog.RequestContent = string(reqJSON)
		}
	}

	// 设置响应内容
	if m.InternalResponse != nil {
		// 创建响应的浅拷贝，过滤掉 images 字段以减少存储压力
		respForLog := m.filterResponseForLog(m.InternalResponse)
		if respJSON, jsonErr := json.Marshal(respForLog); jsonErr == nil {
			// 如果是 Anthropic 响应，补充 cache_creation_input_tokens 字段
			if m.InternalResponse.Usage != nil && m.InternalResponse.Usage.AnthropicUsage {
				respStr := string(respJSON)
				old := `"usage":{`
				insert := fmt.Sprintf(`"usage":{"cache_creation_input_tokens":%d,`, m.InternalResponse.Usage.CacheCreationInputTokens)
				respJSON = []byte(strings.Replace(respStr, old, insert, 1))
			}
			relayLog.ResponseContent = string(respJSON)
		}
	}

	// 设置错误信息
	if err != nil {
		relayLog.Error = err.Error()
	}

	// 设置 Probe 统计信息
	relayLog.ProbeInputTokens = m.Probe.InputTokens
	relayLog.ProbeOutputTokens = m.Probe.OutputTokens
	relayLog.ProbeCost = m.Probe.Cost

	if logErr := op.RelayLogAdd(ctx, relayLog); logErr != nil {
		log.Warnf("failed to save relay log: %v", logErr)
	}
}

func finalChannel(attempts []model.ChannelAttempt, fallbackID int, fallbackName string) (int, string) {
	if len(attempts) == 0 {
		return fallbackID, fallbackName
	}
	for i := len(attempts) - 1; i >= 0; i-- {
		if attempts[i].Success {
			return attempts[i].ChannelID, attempts[i].ChannelName
		}
	}
	last := attempts[len(attempts)-1]
	if last.ChannelID != 0 || last.ChannelName != "" {
		return last.ChannelID, last.ChannelName
	}
	return fallbackID, fallbackName
}

// filterResponseForLog 创建响应的浅拷贝，过滤掉 images 和 MultipleContent 中的图片数据以减少存储压力
func (m *RelayMetrics) filterResponseForLog(resp *transformerModel.InternalLLMResponse) *transformerModel.InternalLLMResponse {
	if resp == nil {
		return nil
	}

	// 创建浅拷贝
	filtered := *resp
	filtered.Choices = make([]transformerModel.Choice, len(resp.Choices))

	for i, choice := range resp.Choices {
		filtered.Choices[i] = choice

		// 处理 Message
		if choice.Message != nil {
			msgCopy := *choice.Message
			// 清除 Images 字段
			if len(msgCopy.Images) > 0 {
				msgCopy.Images = nil
			}
			// 过滤 MultipleContent 中的图片数据
			if len(msgCopy.Content.MultipleContent) > 0 {
				msgCopy.Content = m.filterMessageContent(msgCopy.Content)
			}
			filtered.Choices[i].Message = &msgCopy
		}

		// 处理 Delta
		if choice.Delta != nil {
			deltaCopy := *choice.Delta
			// 清除 Images 字段
			if len(deltaCopy.Images) > 0 {
				deltaCopy.Images = nil
			}
			// 过滤 MultipleContent 中的图片数据
			if len(deltaCopy.Content.MultipleContent) > 0 {
				deltaCopy.Content = m.filterMessageContent(deltaCopy.Content)
			}
			filtered.Choices[i].Delta = &deltaCopy
		}
	}

	return &filtered
}

// filterMessageContent 过滤 MessageContent 中的图片数据
func (m *RelayMetrics) filterMessageContent(content transformerModel.MessageContent) transformerModel.MessageContent {
	if len(content.MultipleContent) == 0 {
		return content
	}

	filteredParts := make([]transformerModel.MessageContentPart, 0, len(content.MultipleContent))
	for _, part := range content.MultipleContent {
		if part.Type == "image_url" && part.ImageURL != nil {
			// 用占位符替换图片数据
			filteredParts = append(filteredParts, transformerModel.MessageContentPart{
				Type: "image_url",
				ImageURL: &transformerModel.ImageURL{
					URL: "[image data omitted for storage]",
				},
			})
		} else {
			filteredParts = append(filteredParts, part)
		}
	}

	return transformerModel.MessageContent{
		Content:         content.Content,
		MultipleContent: filteredParts,
	}
}
