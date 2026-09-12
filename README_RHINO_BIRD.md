# 课题三 · 质量评测基线与成本可观测

> 腾讯犀牛鸟开源人才培养计划 · 实作成果
> 基础仓库：[WeKnora](https://github.com/Tencent/WeKnora)
> 最终 Tag：`rhino-2026-final-三`　完整 Commit：`9a417d39cfb1c5c146858cababc8ae2314119924`

本文件说明**任务一～六完成了什么、如何验证、以及现存的问题与特性**。
（根目录原有的 `README.md` 为 WeKnora 官方文档，未改动。）

---

## 一、课题背景

WeKnora 的评测指标（Precision / Recall / MRR / NDCG / MAP / BLEU / ROUGE）已经齐全，但缺少一套**可复现的端到端评测流程**与**质量回归门禁**；成本侧也缺**结构化沉淀**，embedding 完全没有缓存，重复索引时是纯粹的重复计算。

课题三的目标是补齐这条「**质量可评测 + 成本可观测**」的工程链路。

---

## 二、完成情况总览

| # | 任务 | 状态 | 一句话成果 |
|---|------|------|-----------|
| 1 | 可重复评测流程（结果入库） | ✅ 已完成 | 评测结果写入数据库，重启不丢 |
| 2 | 四类结果（检索/答案/成本/耗时） | ✅ 已完成 | 单次运行同时给出成本与耗时 |
| 3 | CI 评测门禁 | ✅ 已完成 | 指标退化超阈值时阻断合并 |
| 4 | 模型调用落库 + 用量展示 | ✅ 已完成 | 每次调用落库，模型页展示用量/命中率/费用 |
| 5 | embedding 缓存 | ✅ 已完成 | 相同「模型+维度+文本」直接复用，不重复调用 |
| 6 | 提示词顺序优化 | ✅ 已完成 | 固定内容前置、可变内容后置，提高前缀缓存命中率 |
| 7 | 8 个解析引擎横向基线（选做） | ⬜ 未做 | 受外部服务/凭证限制，未实施 |

---

## 三、各任务做了什么

### 任务一：评测结果入库
- **新建数据表** `evaluation_runs`（任务快照：数据集、模型、分块参数、状态、起止时间、样本数）与 `evaluation_metrics`（各项指标分数）。
- **迁移脚本**：`migrations/versioned/000087_evaluation.{up,down}.sql`、`migrations/sqlite/000013_evaluation.{up,down}.sql`。
- **替换存储**：把原先存在**进程内存**里的评测状态改为落库，重启后可查。
- **关键文件**：`internal/application/repository/evaluation.go`、`internal/application/service/evaluation.go`、`internal/types/evaluation.go`。

### 任务二：成本 + 耗时
- 在 `evaluation_metrics` 增加 `cost`（费用）与 `latency_ms`（耗时）两列。
- **费用计算**：`internal/types/pricing.go` 的 `ComputeTokenCost`（内置默认定价），与任务四共用同一口径。
- **关键文件**：`internal/application/service/evaluation.go`、`internal/models/chat/usage.go`。

### 任务三：CI 评测门禁
- **新增工作流** `.github/workflows/evaluation.yml`：定时/手动触发，起环境后跑评测。
- **一条命令可复现**：`scripts/evaluate.sh` + `scripts/ci-setup.sh`。
- **阈值阻断**：`scripts/check-eval-threshold.py` 读取结果 JSON，逐项与 `scripts/eval-thresholds.json` 比较，任一指标低于阈值即 `exit 1`。
- **阈值基线**：`scripts/eval-thresholds.json` 的检索类阈值（precision/recall/ndcg/mrr/map = 0.5）取自默认数据集的一次真实评测（precision=1、recall=0.75、ndcg/mrr/map=1），取 0.5 作安全边际——低于基线、又能拦住"检索退化到全 0"的情况；生成类指标（BLEU/ROUGE）因 LLM 输出非确定，不设门禁（保持 0.0）。若换更大数据集，应按新基线重新校准。
- **注意**：CI 中必须用 `--build` 构建镜像（否则跑的是官方镜像、不是 PR 代码）；fork PR 因 GitHub secrets 限制无法取用评测凭证，属安全机制而非缺陷。

### 任务四：模型调用落库 + 模型页展示
- **新建数据表** `model_usage_records`（`model_id`、租户、`purpose`、prompt/completion/cached token、`cost` 等）。
  迁移：`migrations/versioned/000088_model_usage.*`、`migrations/sqlite/000014_model_usage.*`。
- **落库钩子**：在 `internal/models/chat/usage.go` 的统一出口 `logUsage` 旁挂 `UsageRecorder`，**异步**写入，覆盖所有 chat 类调用，不阻塞主流程。
- **查询接口**：`GET /api/v1/models/usage`（`internal/handler/model.go` + `internal/router/routes_infra.go`），按模型与时间区间聚合。
- **前端面板**：`frontend/src/views/settings/ModelSettings.vue` 新增「用量统计」tab，展示调用次数 / 缓存命中率 / 费用。

### 任务五：embedding 缓存
- **新增缓存装饰器**（实现同一个 `Embedder` 接口，对上游完全透明）：
  - `internal/models/embedding/cache.go`：`EmbeddingCache` 接口 + `cacheEmbedder` 装饰器 + 键函数。
  - `internal/models/embedding/cache_memory.go`：进程内缓存（TTL 24h + 上限 1 万条 + 全局单例）。
- **缓存键** = `SHA256(模型名 + 维度 + 文本)` 前 16 位；维度参与指纹，避免不同维度串用。
- **覆盖三个入口**：`Embed` / `BatchEmbed` / `BatchEmbedWithPool`（重建索引主路径）。
- **挂载点**：`internal/models/embedding/embedder.go` 的 `NewEmbedder` 装饰器链**最外层**——命中即在进入观测/限流/落库之前短路，不产生任何真实调用。

### 任务六：提示词顺序优化
- **核心原则**：固定内容（系统规则）前置、可变内容（检索结果）后置，让厂商前缀缓存能命中。
- **检索结果后置**：默认 system prompt 模板 `default_kb` 移除 `{{contexts}}`，改由默认 context 模板 `default_context` 承载 —— 即把检索结果从 **system prompt** 移到 **user message**。
  - `config/prompt_templates/system_prompt.yaml`、`config/prompt_templates/context_template.yaml`
  - （只改默认模板，用户自定义模板不受影响；渲染代码一行未动。）
- **时间戳降频**：`internal/types/placeholder.go` 的 `{{current_time}}` 由「精确到秒」改为「分钟」，消除"每秒都变"对前缀的破坏。

---

## 四、如何验证

### 4.1 单元测试（最快，无需外部服务）

```bash
# 任务五：embedding 缓存（键唯一、命中拦截、TTL 过期、副本隔离）
go test ./internal/models/embedding/ -run 'TestEmbeddingCacheKey|TestCacheEmbedder|TestMemoryEmbeddingCache' -v

# 任务六：时间戳降频 + 模板顺序
go test ./internal/types/   -run TestRenderPromptPlaceholders -v
go test ./internal/config/  -run TestPromptTemplatesPrefixCacheOrdering -v

# 相关包完整测试
go test ./internal/models/embedding/ ./internal/types/ ./internal/config/ ./internal/database/
```

**通过标志**：全部 `PASS` / `ok`。

### 4.2 端到端验证（需要 docker 环境）

```bash
# 用**你自己改后的代码**构建并启动（--build 是关键，否则跑的是官方镜像）
docker compose up -d --build app
docker ps            # 等待 WeKnora-app 变为 healthy
```

**任务一 / 二：评测结果入库 + 成本耗时**

```bash
# 触发一次评测
curl -s -X POST http://localhost:8080/api/v1/evaluation -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}'

# 重启后端后再查（验证"重启不丢"），并看 cost / latency_ms
docker exec WeKnora-postgres psql -U postgres -d WeKnora -c \
  "SELECT run_id, cost, latency_ms FROM evaluation_metrics ORDER BY created_at DESC LIMIT 3;"
```
**通过标志**：重启后仍能查到结果；`cost > 0` 且 `latency_ms > 0`。

**任务三：CI 门禁**

在 GitHub 上查看 Actions 中 `evaluation.yml` 的运行；故意下调检索阈值提交 PR，CI 应变红并明确报出退化的指标。

**任务四：调用落库 + 用量接口**

```bash
docker exec WeKnora-postgres psql -U postgres -d WeKnora -c \
  "SELECT model_id, purpose, prompt_tokens, completion_tokens, cost FROM model_usage_records ORDER BY created_at DESC LIMIT 10;"

curl -s "http://localhost:8080/api/v1/models/usage" -H "Authorization: Bearer $TOKEN" | jq .
```
**通过标志**：库中有多行记录；接口返回按模型的调用次数、总 token、缓存命中率、费用。
前端：「设置 → 模型配置 → 用量统计」tab 有数据。

**任务五：embedding 缓存**

```bash
# 1) 重启后端清空进程内缓存
docker compose restart app
# 2) 在界面对**同一份文档**执行两次"重建"
# 3) 数两次的真实 embedding 调用次数
docker logs --since 5m WeKnora-app 2>&1 | grep -c "Embedder BatchEmbed"
```
**通过标志**：第二次的调用次数**显著下降**（正文 chunk 命中缓存、不再调 API）。
> 注意：若知识库开启了「问题生成 / 摘要」，这部分内容由 LLM 每次现生成、文本每次不同，其 embedding 无法命中属**预期行为**；要拿到接近 0 的干净数字，可先在知识库设置中关闭问题生成与摘要。

**任务六：前缀缓存命中率**

```bash
docker exec WeKnora-postgres psql -U postgres -d WeKnora -c \
  "SELECT SUM(cached_tokens) AS cached, SUM(prompt_tokens) AS prompt, \
          ROUND(100.0*SUM(cached_tokens)/NULLIF(SUM(prompt_tokens),0),1) AS hit_rate \
   FROM model_usage_records WHERE purpose='knowledge_qa';"
```
**通过标志**：**同一会话连续多轮**提问后，`hit_rate` 相比改动前提升。
> 单轮提问没有可复用前缀，提升不明显——务必用多轮对话对比。

---

## 五、已知问题与遗留特性

### 5.1 设计特性（有意为之，非缺陷）

| # | 特性 | 说明 |
|---|------|------|
| 1 | **embedding 缓存是进程内的** | 进程重启即清空，也不跨多副本共享 → "两次重建对比"必须在**同一次运行内**完成，中途不能重启后端。 |
| 2 | **缓存键含维度** | 换 embedding 模型（维度变）后旧缓存全部失效，需重新向量化。 |
| 3 | **缓存有 TTL 与容量上限** | 默认 TTL 24h、上限 1 万条；超限整体重置（退化为无缓存，不影响正确性）。 |
| 4 | **命中缓存"无痕"** | 命中在最外层短路，不产生日志、不落库、不占并发额度 → 只能通过"调用次数变少"反推。 |
| 5 | **任务六只改默认模板** | 检索结果渲染逻辑同时传给 system prompt 与 user message，本次仅调整**默认模板**的落点；用户自定义模板行为不变。 |

### 5.2 遗留问题（后续可迭代）

| # | 问题 | 影响 |
|---|------|------|
| 1 | **embedding 调用不落库** | 任务四的落库钩子只挂在 **chat 模型**上，`internal/models/embedding/` 未接入 → 数据库里查不到 embedding 调用，成本可观测在此环节有缺口。 |
| 2 | **"问题生成 + 摘要"造成假残留** | 每次重建都用 LLM 现生成，文本每次不同 → 这部分的 embedding 永远无法命中缓存（**不是缓存失效**）。 |
| 3 | **Wiki 生成极度耗时** | 开启 Wiki 的知识库，每传一份文档都会触发整套 Wiki 生成（实测一份 PDF 生成 38 页、耗时约 4 分 39 秒），与 embedding 无关，但会严重拖慢验证。 |
| 4 | **前缀缓存是否生效取决于厂商** | 只有支持「前缀缓存」的厂商（OpenAI / DeepSeek / Gemini 等）才可能命中；任务六只是"创造条件"。 |
| 5 | **system prompt 仍有其他动态项** | 除时间戳外，技能元数据（skills metadata）、知识库绑定信息等也会随会话变化，仍可能破坏前缀——任务六未穷尽所有动态来源。 |
| 6 | **进程内缓存不跨实例/重启** | 多副本或频繁重启的部署下，命中率显著低于单实例长跑。若要跨实例复用，可把 `memoryEmbeddingCache` 换成 Redis 实现（`EmbeddingCache` 接口已抽象好，替换存储即可）。 |

### 5.3 环境/外部依赖类问题（非代码缺陷）

| # | 问题 | 说明 |
|---|------|------|
| 1 | **embedding 依赖外部 API，配额/风控是硬约束** | 实测 Gemini 免费层会返回 `403 PERMISSION_DENIED（Your project has been denied access）`——项目级风控。建议验证时优先选用国内可直连、配额稳定的 embedding（如硅基流动、智谱）。 |
| 2 | **文档后处理耗时与文件大小无关** | 耗时主要在 embedding + 摘要 + 问题生成 + Wiki 生成（均为远程调用）。一个 1.2 KB 的 txt 也可能因开启 Wiki 而耗时数分钟。排查"卡住"不应以文件大小为依据。 |
| 3 | **前端状态刷新滞后** | 后端已 `completed` 时前端仍可能显示"处理中"，需手动刷新；排查应以数据库 `knowledges.parse_status` 为准。 |
| 4 | **docker 本地构建与宿主机 Go 缓存隔离** | `docker compose --build` 在隔离环境内重新下载全部依赖（实测首次可超 1600 秒），与宿主机 `~/go/pkg/mod` 不共享；国内网络下需设 `GOPROXY_ARG` 指向国内镜像源。 |

---

## 六、未完成项

- **任务七（选做）**：8 个解析引擎横向基线。实际可开箱运行的引擎只有 `simple` / `builtin` / `anydoc` 三个；其余 5 个（`weknoracloud` / `mineru` / `mineru_cloud` / `paddleocr_vl` / `paddleocr_vl_cloud`）需要部署额外服务或申请云凭证，受外部资源限制**未实施**。

---

## 附：本次改动文件速查

```
迁移：
  migrations/versioned/000087_evaluation.*   migrations/sqlite/000013_evaluation.*
  migrations/versioned/000088_model_usage.*  migrations/sqlite/000014_model_usage.*

后端：
  internal/types/evaluation.go                internal/types/model_usage.go
  internal/types/pricing.go                   internal/types/placeholder.go
  internal/application/repository/evaluation.go
  internal/application/repository/model_usage_record.go
  internal/application/service/evaluation.go
  internal/models/chat/usage.go
  internal/models/embedding/cache.go          internal/models/embedding/cache_memory.go
  internal/models/embedding/embedder.go
  internal/handler/model.go                   internal/router/routes_infra.go
  internal/container/container.go

CI / 脚本：
  .github/workflows/evaluation.yml
  scripts/evaluate.sh  scripts/check-eval-threshold.py  scripts/eval-thresholds.json  scripts/ci-setup.sh

前端：
  frontend/src/api/model/index.ts
  frontend/src/views/settings/ModelSettings.vue
  frontend/src/i18n/locales/{zh-CN,en-US,ko-KR,ru-RU}.ts

模板：
  config/prompt_templates/system_prompt.yaml  config/prompt_templates/context_template.yaml

测试（新增）：
  internal/models/embedding/cache_test.go
  internal/types/placeholder_test.go
  internal/config/prompt_template_cache_test.go
```
