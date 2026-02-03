package helper

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/dlclark/regexp2"
)

func FetchModels(ctx context.Context, request model.Channel) ([]string, error) {
	client, err := ChannelHttpClient(&request)
	if err != nil {
		return nil, err
	}
	fetchModel := make([]string, 0)
	switch request.Type {
	case outbound.OutboundTypeAnthropic:
		fetchModel, err = fetchAnthropicModels(client, ctx, request)
	case outbound.OutboundTypeGemini:
		fetchModel, err = fetchGeminiModels(client, ctx, request)
	default:
		fetchModel, err = fetchOpenAIModels(client, ctx, request)
	}
	if err != nil {
		return nil, err
	}
	if request.MatchRegex != nil && *request.MatchRegex != "" {
		matchModel := make([]string, 0)
		re, err := regexp2.Compile(*request.MatchRegex, regexp2.ECMAScript)
		if err != nil {
			return nil, err
		}
		for _, model := range fetchModel {
			matched, err := re.MatchString(model)
			if err != nil {
				return nil, err
			}
			if matched {
				matchModel = append(matchModel, model)
			}
		}
		return matchModel, nil
	}
	return fetchModel, nil
}

// refer: https://platform.openai.com/docs/api-reference/models/list
func fetchOpenAIModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	keys := request.GetAllEnabledKeys()
	if len(keys) == 0 {
		return nil, nil
	}

	modelSet := make(map[string]struct{})
	var lastErr error

	for _, key := range keys {
		req, _ := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			request.GetBaseUrl()+"/models",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+key.ChannelKey)
		for _, header := range request.CustomHeader {
			if header.HeaderKey != "" {
				req.Header.Set(header.HeaderKey, header.HeaderValue)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		var result model.OpenAIModelList
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body.Close()

		for _, m := range result.Data {
			modelSet[m.ID] = struct{}{}
		}
	}

	if len(modelSet) == 0 && lastErr != nil {
		return nil, lastErr
	}

	models := make([]string, 0, len(modelSet))
	for m := range modelSet {
		models = append(models, m)
	}
	return models, nil
}

// refer: https://ai.google.dev/api/models
func fetchGeminiModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	keys := request.GetAllEnabledKeys()
	if len(keys) == 0 {
		return nil, nil
	}

	modelSet := make(map[string]struct{})
	var lastErr error

	for _, key := range keys {
		pageToken := ""
		for {
			req, _ := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				request.GetBaseUrl()+"/models",
				nil,
			)
			req.Header.Set("X-Goog-Api-Key", key.ChannelKey)
			for _, header := range request.CustomHeader {
				if header.HeaderKey != "" {
					req.Header.Set(header.HeaderKey, header.HeaderValue)
				}
			}
			if pageToken != "" {
				q := req.URL.Query()
				q.Add("pageToken", pageToken)
				req.URL.RawQuery = q.Encode()
			}

			resp, err := client.Do(req)
			if err != nil {
				lastErr = err
				break
			}

			var result model.GeminiModelList
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				resp.Body.Close()
				lastErr = err
				break
			}
			resp.Body.Close()

			for _, m := range result.Models {
				name := strings.TrimPrefix(m.Name, "models/")
				modelSet[name] = struct{}{}
			}

			if result.NextPageToken == "" {
				break
			}
			pageToken = result.NextPageToken
		}
	}

	if len(modelSet) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return fetchOpenAIModels(client, ctx, request)
	}

	models := make([]string, 0, len(modelSet))
	for m := range modelSet {
		models = append(models, m)
	}
	return models, nil
}

// refer: https://platform.claude.com/docs
func fetchAnthropicModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	keys := request.GetAllEnabledKeys()
	if len(keys) == 0 {
		return nil, nil
	}

	modelSet := make(map[string]struct{})
	var lastErr error

	for _, key := range keys {
		var afterID string
		for {
			req, _ := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				request.GetBaseUrl()+"/models",
				nil,
			)
			req.Header.Set("X-Api-Key", key.ChannelKey)
			req.Header.Set("Anthropic-Version", "2023-06-01")
			for _, header := range request.CustomHeader {
				if header.HeaderKey != "" {
					req.Header.Set(header.HeaderKey, header.HeaderValue)
				}
			}
			// 设置多页参数
			q := req.URL.Query()
			if afterID != "" {
				q.Set("after_id", afterID)
			}
			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				lastErr = err
				break
			}

			var result model.AnthropicModelList
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				resp.Body.Close()
				lastErr = err
				break
			}
			resp.Body.Close()

			for _, m := range result.Data {
				modelSet[m.ID] = struct{}{}
			}

			if !result.HasMore {
				break
			}
			afterID = result.LastID
		}
	}

	if len(modelSet) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return fetchOpenAIModels(client, ctx, request)
	}

	models := make([]string, 0, len(modelSet))
	for m := range modelSet {
		models = append(models, m)
	}
	return models, nil
}
