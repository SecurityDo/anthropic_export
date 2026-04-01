# plugin_Anthropic

Exports Anthropic logs and telemetry data for SIEM ingestion via the [Admin API](https://docs.anthropic.com/en/api/admin).

## Prerequisites

- An **Admin API key** (starts with `sk-ant-admin...`), provisioned by an organization admin in the [Claude Console](https://console.anthropic.com). A regular API key (`sk-ant-api03-...`) will **not** work — the usage and cost report endpoints are Admin-only.

## Build

```bash
go build -o anthropic-export ./cmd/anthropic_export/
```

## Usage

```bash
# All reports, last 7 days
./anthropic-export --api-key sk-ant-admin-xxxx

# Messages usage only, last 30 days
./anthropic-export --api-key sk-ant-admin-xxxx --days 30 --report messages

# Cost report only, pipe JSONL to file
./anthropic-export --api-key sk-ant-admin-xxxx --report cost > cost.jsonl

# Using environment variable
export ANTHROPIC_ADMIN_API_KEY=sk-ant-admin-xxxx
./anthropic-export --report claude_code
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api-key` | `$ANTHROPIC_ADMIN_API_KEY` | Admin API key |
| `--days` | `7` | Number of days to look back |
| `--report` | `all` | Report type: `messages`, `cost`, `claude_code`, or `all` |

## Reports

| Report | API Endpoint | Data |
|--------|-------------|------|
| `messages` | `GET /v1/organizations/usage_report/messages` | Token usage by model, API key, workspace |
| `cost` | `GET /v1/organizations/cost_report` | Daily cost breakdowns (USD) |
| `claude_code` | `GET /v1/organizations/usage_report/claude_code` | Per-user Claude Code sessions, commits, PRs, lines changed |

## Output Format

JSONL (one JSON object per line) on **stdout** -- the standard format for SIEM ingestion (Splunk, Elastic, Sentinel, etc.). Progress messages go to stderr.

```jsonl
{"event_type":"anthropic.messages_usage","timestamp":"2026-03-25T00:00:00Z","data":{"model":"claude-sonnet-4-20250514","uncached_input_tokens":12345,"output_tokens":5000,...}}
{"event_type":"anthropic.cost","timestamp":"2026-03-25T00:00:00Z","data":{"amount":"1.23","currency":"USD","cost_type":"tokens","model":"claude-sonnet-4-20250514",...}}
```
