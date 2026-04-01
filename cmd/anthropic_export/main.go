package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	baseURL        = "https://api.anthropic.com"
	anthropicVer   = "2023-06-01"
	defaultLimit   = 100
)

// ---- response types ----

// Messages usage report

type MessagesUsageReport struct {
	Data    []MessagesUsageBucket `json:"data"`
	HasMore bool                  `json:"has_more"`
	NextPage string               `json:"next_page"`
}

type MessagesUsageBucket struct {
	StartingAt string                 `json:"starting_at"`
	EndingAt   string                 `json:"ending_at"`
	Results    []MessagesUsageResult  `json:"results"`
}

type MessagesUsageResult struct {
	APIKeyID             *string          `json:"api_key_id"`
	WorkspaceID          *string          `json:"workspace_id"`
	Model                *string          `json:"model"`
	ServiceTier          string           `json:"service_tier"`
	ContextWindow        string           `json:"context_window"`
	InferenceGeo         *string          `json:"inference_geo"`
	Speed                *string          `json:"speed"`
	UncachedInputTokens  int64            `json:"uncached_input_tokens"`
	CacheReadInputTokens int64            `json:"cache_read_input_tokens"`
	CacheCreation        *CacheCreation   `json:"cache_creation"`
	OutputTokens         int64            `json:"output_tokens"`
	ServerToolUse        *ServerToolUse   `json:"server_tool_use"`
}

type CacheCreation struct {
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
}

type ServerToolUse struct {
	WebSearchRequests int64 `json:"web_search_requests"`
}

// Cost report

type CostReport struct {
	Data    []CostBucket `json:"data"`
	HasMore bool         `json:"has_more"`
	NextPage string      `json:"next_page"`
}

type CostBucket struct {
	StartingAt string       `json:"starting_at"`
	EndingAt   string       `json:"ending_at"`
	Results    []CostResult `json:"results"`
}

type CostResult struct {
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	CostType      string  `json:"cost_type"`
	WorkspaceID   *string `json:"workspace_id"`
	Description   *string `json:"description"`
	Model         *string `json:"model"`
	ServiceTier   string  `json:"service_tier"`
	TokenType     string  `json:"token_type"`
	ContextWindow string  `json:"context_window"`
	InferenceGeo  *string `json:"inference_geo"`
	Speed         *string `json:"speed"`
}

// Claude Code usage report

type ClaudeCodeUsageReport struct {
	Data    []ClaudeCodeUsageEntry `json:"data"`
	HasMore bool                   `json:"has_more"`
	NextPage string                `json:"next_page"`
}

type ClaudeCodeUsageEntry struct {
	Date       string          `json:"date"`
	UserID     *string         `json:"user_id"`
	APIKeyID   *string         `json:"api_key_id"`
	Sessions   int64           `json:"sessions"`
	Commits    int64           `json:"commits"`
	PRs        int64           `json:"prs"`
	LinesAdded int64           `json:"lines_added"`
	LinesRemoved int64         `json:"lines_removed"`
	TokenUsage json.RawMessage `json:"token_usage"`
}

// Organization info

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ---- API client ----

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(path string, params url.Values) ([]byte, int, error) {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVer)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// GetOrganization returns info about the authenticated organization.
func (c *Client) GetOrganization() (*Organization, error) {
	body, status, err := c.get("/v1/organizations/me", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("GET /v1/organizations/me returned %d: %s", status, body)
	}
	var org Organization
	if err := json.Unmarshal(body, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// GetMessagesUsage fetches one page of the messages usage report.
func (c *Client) GetMessagesUsage(startingAt, endingAt, bucketWidth, page string, groupBy []string) (*MessagesUsageReport, error) {
	params := url.Values{}
	params.Set("starting_at", startingAt)
	if endingAt != "" {
		params.Set("ending_at", endingAt)
	}
	if bucketWidth != "" {
		params.Set("bucket_width", bucketWidth)
	}
	if page != "" {
		params.Set("page", page)
	}
	for _, g := range groupBy {
		params.Add("group_by[]", g)
	}

	body, status, err := c.get("/v1/organizations/usage_report/messages", params)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("messages usage API returned %d: %s", status, body)
	}
	var report MessagesUsageReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// GetCostReport fetches one page of the cost report.
func (c *Client) GetCostReport(startingAt, endingAt, page string) (*CostReport, error) {
	params := url.Values{}
	params.Set("starting_at", startingAt)
	if endingAt != "" {
		params.Set("ending_at", endingAt)
	}
	params.Set("bucket_width", "1d")
	if page != "" {
		params.Set("page", page)
	}

	body, status, err := c.get("/v1/organizations/cost_report", params)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("cost report API returned %d: %s", status, body)
	}
	var report CostReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// GetClaudeCodeUsage fetches one page of Claude Code usage for a single day.
func (c *Client) GetClaudeCodeUsage(date, page string) (*ClaudeCodeUsageReport, error) {
	params := url.Values{}
	params.Set("starting_at", date)
	params.Set("limit", fmt.Sprintf("%d", defaultLimit))
	if page != "" {
		params.Set("page", page)
	}

	body, status, err := c.get("/v1/organizations/usage_report/claude_code", params)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("claude code usage API returned %d: %s", status, body)
	}
	var report ClaudeCodeUsageReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// ---- helpers ----

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// printJSONLines outputs each item as a single JSON line (JSONL), the standard
// format consumed by SIEM products like Splunk, Elastic, Sentinel, etc.
func printJSONLines(eventType string, timestamp string, records any) {
	raw, _ := json.Marshal(records)
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		// not an array — emit as a single event
		envelope := map[string]any{
			"event_type": eventType,
			"timestamp":  timestamp,
			"data":       json.RawMessage(raw),
		}
		fmt.Println(mustJSON(envelope))
		return
	}
	for _, item := range items {
		envelope := map[string]any{
			"event_type": eventType,
			"timestamp":  timestamp,
			"data":       item,
		}
		fmt.Println(mustJSON(envelope))
	}
}

// ---- main ----

func main() {
	apiKey := flag.String("api-key", "", "Anthropic Admin API key (sk-ant-admin...)")
	days := flag.Int("days", 7, "Number of days to look back")
	reportType := flag.String("report", "all", "Report type: messages, cost, claude_code, or all")
	flag.Parse()

	if *apiKey == "" {
		// fall back to environment variable
		*apiKey = os.Getenv("ANTHROPIC_ADMIN_API_KEY")
	}
	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: provide --api-key or set ANTHROPIC_ADMIN_API_KEY")
		os.Exit(1)
	}

	client := NewClient(*apiKey)

	// Verify connectivity by fetching org info
	org, err := client.GetOrganization()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching organization info: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Connected to organization: %s (%s)\n", org.Name, org.ID)

	now := time.Now().UTC()
	startTime := now.AddDate(0, 0, -*days).Truncate(24 * time.Hour)
	endTime := now.Truncate(24 * time.Hour).Add(24 * time.Hour) // end of today
	startStr := startTime.Format(time.RFC3339)
	endStr := endTime.Format(time.RFC3339)

	fmt.Fprintf(os.Stderr, "Time range: %s to %s\n", startStr, endStr)

	// --- Messages Usage ---
	if *reportType == "all" || *reportType == "messages" {
		fmt.Fprintln(os.Stderr, "\n--- Fetching Messages Usage Report ---")
		page := ""
		totalBuckets := 0
		for {
			report, err := client.GetMessagesUsage(
				startStr, endStr, "1d", page,
				[]string{"model", "api_key_id"},
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				break
			}
			for _, bucket := range report.Data {
				totalBuckets++
				printJSONLines("anthropic.messages_usage", bucket.StartingAt, bucket.Results)
			}
			if !report.HasMore || report.NextPage == "" {
				break
			}
			page = report.NextPage
		}
		fmt.Fprintf(os.Stderr, "Messages usage: %d time buckets exported\n", totalBuckets)
	}

	// --- Cost Report ---
	if *reportType == "all" || *reportType == "cost" {
		fmt.Fprintln(os.Stderr, "\n--- Fetching Cost Report ---")
		page := ""
		totalBuckets := 0
		for {
			report, err := client.GetCostReport(startStr, endStr, page)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				break
			}
			for _, bucket := range report.Data {
				totalBuckets++
				printJSONLines("anthropic.cost", bucket.StartingAt, bucket.Results)
			}
			if !report.HasMore || report.NextPage == "" {
				break
			}
			page = report.NextPage
		}
		fmt.Fprintf(os.Stderr, "Cost report: %d time buckets exported\n", totalBuckets)
	}

	// --- Claude Code Usage ---
	if *reportType == "all" || *reportType == "claude_code" {
		fmt.Fprintln(os.Stderr, "\n--- Fetching Claude Code Usage Report ---")
		totalEntries := 0
		// Iterate day by day since this endpoint takes a single date
		for d := startTime; d.Before(endTime); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			page := ""
			for {
				report, err := client.GetClaudeCodeUsage(dateStr, page)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  %s: %v\n", dateStr, err)
					break
				}
				for _, entry := range report.Data {
					totalEntries++
					printJSONLines("anthropic.claude_code_usage", dateStr, []ClaudeCodeUsageEntry{entry})
				}
				if !report.HasMore || report.NextPage == "" {
					break
				}
				page = report.NextPage
			}
		}
		fmt.Fprintf(os.Stderr, "Claude Code usage: %d entries exported\n", totalEntries)
	}

	fmt.Fprintln(os.Stderr, "\nDone. JSONL output written to stdout.")
}
