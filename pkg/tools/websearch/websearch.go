package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/strin/unlimitedclaw/pkg/tools"
)

const (
	userAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	searchTimeout     = 10 * time.Second
	defaultMaxResults = 5
)

var (
	// DuckDuckGo result extraction patterns
	reDDGLink    = regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	reDDGSnippet = regexp.MustCompile(`<a class="result__snippet[^"]*".*?>([\s\S]*?)</a>`)
	reTags       = regexp.MustCompile(`<[^>]+>`)
)

// WebSearchTool implements web search using DuckDuckGo
type WebSearchTool struct {
	client     *http.Client
	maxResults int
}

// Option is a functional option for WebSearchTool
type Option func(*WebSearchTool)

// WithHTTPClient sets a custom HTTP client (for testing)
func WithHTTPClient(client *http.Client) Option {
	return func(t *WebSearchTool) {
		t.client = client
	}
}

// WithMaxResults sets the maximum number of results to return
func WithMaxResults(n int) Option {
	return func(t *WebSearchTool) {
		if n > 0 {
			t.maxResults = n
		}
	}
}

// New creates a new WebSearchTool with the given options
func New(opts ...Option) *WebSearchTool {
	t := &WebSearchTool{
		client: &http.Client{
			Timeout: searchTimeout,
		},
		maxResults: defaultMaxResults,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Name implements tools.Tool
func (t *WebSearchTool) Name() string {
	return "web_search"
}

// Description implements tools.Tool
func (t *WebSearchTool) Description() string {
	return "Search the web for information using DuckDuckGo"
}

// Parameters implements tools.Tool
func (t *WebSearchTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

// Execute implements tools.Tool
func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &tools.ToolResult{
			ForLLM:  "Error: 'query' parameter is required and must be a non-empty string",
			IsError: true,
		}, fmt.Errorf("missing or invalid query parameter")
	}

	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Search failed: failed to create request: %v", err),
			IsError: true,
		}, err
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("Search failed: %v", ctx.Err()),
				IsError: true,
			}, ctx.Err()
		}
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Search failed: %v", err),
			IsError: true,
		}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Search failed: HTTP %d", resp.StatusCode),
			IsError: true,
		}, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Search failed: failed to read response: %v", err),
			IsError: true,
		}, err
	}

	forLLM, forUser := t.extractResults(string(body), query)

	return &tools.ToolResult{
		ForLLM:  forLLM,
		ForUser: forUser,
		IsError: false,
	}, nil
}

// extractResults parses HTML and extracts search results
func (t *WebSearchTool) extractResults(html, query string) (forLLM, forUser string) {
	linkMatches := reDDGLink.FindAllStringSubmatch(html, t.maxResults+5)

	if len(linkMatches) == 0 {
		msg := fmt.Sprintf("No results found for: %s", query)
		return msg, msg
	}

	snippetMatches := reDDGSnippet.FindAllStringSubmatch(html, t.maxResults+5)

	maxItems := len(linkMatches)
	if maxItems > t.maxResults {
		maxItems = t.maxResults
	}

	var llmLines []string
	for i := 0; i < maxItems; i++ {
		urlStr := linkMatches[i][1]
		title := stripTags(linkMatches[i][2])
		title = strings.TrimSpace(title)

		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				_, after, ok := strings.Cut(u, "uddg=")
				if ok {
					urlStr = after
				}
			}
		}

		llmLines = append(llmLines, fmt.Sprintf("%d. [%s](%s)", i+1, title, urlStr))

		if i < len(snippetMatches) {
			snippet := stripTags(snippetMatches[i][1])
			snippet = strings.TrimSpace(snippet)
			if snippet != "" {
				llmLines = append(llmLines, snippet)
			}
		}
		llmLines = append(llmLines, "")
	}

	var userLines []string
	userLines = append(userLines, fmt.Sprintf("🔍 Results for '%s':", query))
	for i := 0; i < maxItems; i++ {
		urlStr := linkMatches[i][1]
		title := stripTags(linkMatches[i][2])
		title = strings.TrimSpace(title)

		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				_, after, ok := strings.Cut(u, "uddg=")
				if ok {
					urlStr = after
				}
			}
		}

		userLines = append(userLines, fmt.Sprintf("• %s - %s", title, urlStr))
	}

	return strings.Join(llmLines, "\n"), strings.Join(userLines, "\n")
}

// stripTags removes HTML tags from a string
func stripTags(content string) string {
	return reTags.ReplaceAllString(content, "")
}
