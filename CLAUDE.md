# gstack

For all web browsing, use the `/browse` skill from gstack. Never use `mcp__claude-in-chrome__*` tools.

## Available gstack skills

/office-hours, /plan-ceo-review, /plan-eng-review, /plan-design-review, /design-consultation, /review, /ship, /land-and-deploy, /canary, /benchmark, /browse, /qa, /qa-only, /design-review, /setup-browser-cookies, /setup-deploy, /retro, /investigate, /document-release, /codex, /cso, /autoplan, /careful, /freeze, /guard, /unfreeze, /gstack-upgrade

## Setup for teammates

gstack is already installed in this project at `.claude/skills/gstack/`.

To install gstack locally, run:

```bash
git clone https://github.com/garrytan/gstack.git ~/.claude/skills/gstack && cd ~/.claude/skills/gstack && ./setup
```

## ServerMind Project

ServerMind 是一个 AI 驱动的自动化服务器部署平台。

### 核心功能
- **GitHub 项目部署**: 输入 GitHub URL，AI 自动分析并部署
- **教程部署**: 支持百度教程、CSDN、掘金等教程链接
- **Multi-Agent 系统**: 需求解析、代码分析、部署执行、故障诊断
- **RAG 知识库**: 自动积累部署经验

### 快速开始
```bash
# 启动基础服务
docker compose up -d postgres redis

# 启动 API 服务器（需要 Go 1.21+）
go run cmd/server/main.go
```

详细文档见 `docs/DEPLOY.md`
