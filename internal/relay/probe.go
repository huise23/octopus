package relay

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/bestruirui/octopus/internal/utils/log"
)

var channelRetryRunning sync.Map // channelID -> struct{}

func tryLockChannelRetry(channelID int) bool {
	_, loaded := channelRetryRunning.LoadOrStore(channelID, struct{}{})
	return !loaded
}

func unlockChannelRetry(channelID int) {
	channelRetryRunning.Delete(channelID)
}

func truncateTo1K(s string) string {
	rs := []rune(s)
	if len(rs) <= 1000 {
		return s
	}
	return string(rs[:1000])
}

// buildProbeRequest 使用当前请求参数，但截断文本前 1000 字以减少 token 消耗
func buildProbeRequest(orig *model.InternalLLMRequest) *model.InternalLLMRequest {
	clone := *orig

	for i := range clone.Messages {
		msg := &clone.Messages[i]
		if msg.Content.Content != nil {
			truncated := truncateTo1K(*msg.Content.Content)
			msg.Content.Content = &truncated
		}
		for j := range msg.Content.MultipleContent {
			if msg.Content.MultipleContent[j].Type == "text" && msg.Content.MultipleContent[j].Text != nil {
				truncated := truncateTo1K(*msg.Content.MultipleContent[j].Text)
				msg.Content.MultipleContent[j].Text = &truncated
			}
		}
	}

	return &clone
}

// doProbeRequest 用现有 outbound 流程做一次"隐藏"的探测请求，返回 Usage 用于统计 token
func doProbeRequest(
	ctx context.Context,
	ch *dbmodel.Channel,
	key *dbmodel.ChannelKey,
	req *model.InternalLLMRequest,
) (*model.Usage, int, error) {
	outAdapter := outbound.Get(ch.Type)
	if outAdapter == nil {
		return nil, 0, fmt.Errorf("unsupported channel type: %d", ch.Type)
	}

	outboundReq, err := outAdapter.TransformRequest(
		ctx,
		req,
		ch.GetBaseUrl(),
		key.ChannelKey,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create probe request: %w", err)
	}

	httpClient, err := helper.ChannelHttpClient(ch)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get http client: %w", err)
	}

	resp, err := httpClient.Do(outboundReq)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send probe request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, resp.StatusCode, fmt.Errorf("probe upstream error: %d: %s", resp.StatusCode, string(body))
	}

	internalResp, err := outAdapter.TransformResponse(ctx, resp)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to transform probe response: %w", err)
	}
	if internalResp == nil || internalResp.Usage == nil {
		return nil, resp.StatusCode, nil
	}
	return internalResp.Usage, resp.StatusCode, nil
}

// RetryUnavailableForChannel 同一 channel 级别的后台重试协程
// - 同一 channel 同时只运行一个
// - 遍历 channel.Keys，对符合 ShouldRetry 的 key 串行重试
// - 每个重试超时 30 秒
// - 成功/失败记录到熔断器，并在 metrics.Probe 中记录 token/费用
func RetryUnavailableForChannel(
	channel *dbmodel.Channel,
	req *model.InternalLLMRequest,
	metrics *RelayMetrics,
) {
	if !tryLockChannelRetry(channel.ID) {
		return
	}

	go func() {
		defer unlockChannelRetry(channel.ID)

		ctx := context.Background()

		ch, err := op.ChannelGet(channel.ID, ctx)
		if err != nil || !ch.Enabled {
			log.Warnf("RetryUnavailableForChannel: channel %d not available: %v", channel.ID, err)
			return
		}

		probeReq := buildProbeRequest(req)

		for i := range ch.Keys {
			key := &ch.Keys[i]
			if !key.Enabled {
				continue
			}
			if !balancer.ShouldRetry(ch.ID, key.ID, probeReq.Model) {
				continue
			}

			probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			usage, statusCode, err := doProbeRequest(probeCtx, ch, key, probeReq)
			cancel()

			if err == nil {
				balancer.RecordSuccess(ch.ID, key.ID, probeReq.Model)
				persistCircuitState(context.Background(), ch.ID, key.ID, probeReq.Model)
				if usage != nil && metrics != nil {
					metrics.Probe.InputTokens += int(usage.PromptTokens)
					metrics.Probe.OutputTokens += int(usage.CompletionTokens)
					if p := price.GetLLMPrice(metrics.ActualModel); p != nil {
						metrics.Probe.Cost +=
							float64(usage.PromptTokens)*p.Input*1e-6 +
								float64(usage.CompletionTokens)*p.Output*1e-6
					}
					metrics.Probe.Count++
				}
			} else {
				log.Warnf("probe for channel %d key %d failed: %v", ch.ID, key.ID, err)
				balancer.RecordFailureWithStatus(ch.ID, key.ID, probeReq.Model, statusCode)
				persistCircuitState(context.Background(), ch.ID, key.ID, probeReq.Model)
			}
		}
	}()
}
