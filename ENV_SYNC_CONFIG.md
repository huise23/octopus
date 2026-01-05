# 环境变量同步功能配置说明

## 功能概述

当保存渠道时，如果未勾选"使用代理"选项，系统会自动将渠道的根域名同步到配置的环境变量中。

## 配置方式

**使用 Docker 环境变量配置：**

```yaml
version: '3'
services:
  octopus:
    image: your-octopus-image
    environment:
      - OCTOPUS_ENV_SYNC_API_URL=https://your-api-endpoint.com
    # 其他环境变量...
```

或在 `docker run` 命令中：

```bash
docker run -d \
  -e OCTOPUS_ENV_SYNC_API_URL=https://your-api-endpoint.com \
  your-octopus-image
```

## 环境变量说明

| 环境变量 | 类型 | 必填 | 说明 |
|---------|------|------|------|
| `OCTOPUS_ENV_SYNC_API_URL` | string | 否 | 外部 API 的 URL 地址，未设置则不启用同步功能 |

**配置说明：**
- **超时时间**：固定为 5 秒，无需配置
- **启用/禁用**：通过是否设置环境变量控制
- **未设置环境变量**：功能自动禁用，不影响渠道保存

## 使用示例

### 1. 启用功能

设置环境变量：

```bash
export OCTOPUS_ENV_SYNC_API_URL=https://your-proxy-api.com/sync
```

或在 Docker Compose 中：

```yaml
services:
  octopus:
    environment:
      - OCTOPUS_ENV_SYNC_API_URL=https://your-proxy-api.com/sync
```

### 2. 禁用功能

不设置环境变量或设置为空：

```bash
# 不设置环境变量
# 或
export OCTOPUS_ENV_SYNC_API_URL=""
```

### 3. API 调用格式

当渠道未勾选"使用代理"时，系统会发送以下请求：

```bash
curl 'https://your-api-endpoint.com' \
  -X 'PUT' \
  --data-raw 'DOMAIN-SUFFIX,example.com,Proxy'
```

## 工作流程

1. 用户在界面中创建或编辑渠道
2. 未勾选"使用代理"选项
3. 保存渠道时触发环境同步
4. 系统从渠道的 BaseURL 中提取根域名
5. 异步调用外部 API 更新代理规则（5 秒超时）
6. 记录操作日志

## 域名提取示例

| 渠道 BaseURL | 提取的根域名 |
|-------------|-------------|
| `https://api.example.com` | `example.com` |
| `https://v1.api.example.com` | `example.com` |
| `example.com` | `example.com` |
| `https://api.co.uk` | `api.co.uk` |

## 错误处理

- **环境变量未设置**：功能自动禁用，不执行任何操作
- API 调用失败时，**不会**影响渠道的保存
- 所有错误都会记录到日志中
- 日志级别：
  - 成功：INFO
  - API 返回非 200 状态码：WARN
  - 其他错误：ERROR
  - 环境变量未设置：DEBUG

## 注意事项

1. **环境变量优先**：通过环境变量配置，无需修改配置文件
2. **异步执行**：API 调用在后台异步执行，不会阻塞响应
3. **固定超时**：超时时间固定为 5 秒，不可配置
4. **幂等性**：确保外部 API 支持幂等操作
5. **安全性**：
   - 建议使用 HTTPS
   - 考虑添加认证机制（如 API Key）
   - 不要在日志中暴露敏感信息

## 日志示例

**成功同步：**
```
2026-01-05T16:57:04+08:00  INFO Syncing domain api.example.com (root: example.com) to https://your-api.com
2026-01-05T16:57:04+08:00  INFO Successfully synced domain example.com (status: 200)
```

**环境变量未设置：**
```
2026-01-05T16:57:04+08:00  DEBUG Environment variable OCTOPUS_ENV_SYNC_API_URL is not set, skipping sync
```

**API 返回错误：**
```
2026-01-05T16:57:04+08:00  INFO Syncing domain api.example.com (root: example.com) to https://your-api.com
2026-01-05T16:57:04+08:00  WARN EnvSync API returned non-success status 500 for domain example.com
```

## 测试验证

运行测试验证功能：

```bash
# 测试域名提取
cd internal/utils && go test -v

# 测试环境同步服务
cd internal/services && go test -v

# 手动测试环境变量
export OCTOPUS_ENV_SYNC_API_URL=https://test-api.com
cd internal/services && go test -v -run TestNewEnvSyncServiceWithEnv
```

## Docker 部署示例

**docker-compose.yml：**
```yaml
version: '3.8'

services:
  octopus:
    image: your-octopus-image:latest
    container_name: octopus
    ports:
      - "8080:8080"
    environment:
      # 启用环境变量同步功能
      - OCTOPUS_ENV_SYNC_API_URL=https://your-proxy-api.com/sync
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

**Dockerfile：**
```dockerfile
FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o octopus

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/octopus .
COPY --from=builder /app/web ./web
COPY --from=builder /app/data ./data

# 环境变量在运行时传入
# ENV OCTOPUS_ENV_SYNC_API_URL=https://your-api.com

ENTRYPOINT ["./octopus"]
```
