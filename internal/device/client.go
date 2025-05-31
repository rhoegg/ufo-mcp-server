package device

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client handles HTTP communication with the UFO device
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new UFO device client
func NewClient() *Client {
	ufoIP := os.Getenv("UFO_IP")
	if ufoIP == "" {
		ufoIP = "ufo" // default hostname
	}

	return &Client{
		baseURL: fmt.Sprintf("http://%s", ufoIP),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendRawQuery sends a raw query string to the UFO /api endpoint
func (c *Client) SendRawQuery(ctx context.Context, query string) (string, error) {
	// Ensure query doesn't start with ? or /
	if query != "" && (query[0] == '?' || query[0] == '/') {
		query = query[1:]
	}

	url := fmt.Sprintf("%s/api?%s", c.baseURL, query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("UFO request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("UFO returned status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// GetStatus retrieves the current UFO status
func (c *Client) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	// The UFO typically returns JSON status on /api with no parameters
	resp, err := c.SendRawQuery(ctx, "")
	if err != nil {
		return nil, err
	}

	// For now, return a simple map - in a real implementation we'd parse JSON
	status := map[string]interface{}{
		"response":  resp,
		"timestamp": time.Now().Unix(),
	}

	return status, nil
}

// SetRingPattern sends a ring pattern command to the UFO
func (c *Client) SetRingPattern(ctx context.Context, ring string, segments []string, background string, whirlMs int, counterClockwise bool, morphSpec string) error {
	// Build query manually to avoid URL encoding of pipe characters
	var queryParts []string

	// Add ring init
	if ring != "" {
		queryParts = append(queryParts, fmt.Sprintf("%s_init=1", ring))
	}

	// Add segments - join them with | and use the ring name directly
	if len(segments) > 0 {
		segmentStr := ""
		for i, segment := range segments {
			if segment != "" {
				if i > 0 {
					segmentStr += "|"
				}
				segmentStr += segment
			}
		}
		if segmentStr != "" {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", ring, segmentStr))
		}
	}

	// Add background
	if background != "" {
		queryParts = append(queryParts, fmt.Sprintf("%s_bg=%s", ring, background))
	}

	// Add whirl with optional counter-clockwise rotation
	if whirlMs > 0 {
		whirlValue := fmt.Sprintf("%d", whirlMs)
		if counterClockwise {
			whirlValue += "|ccw"
		}
		queryParts = append(queryParts, fmt.Sprintf("%s_whirl=%s", ring, whirlValue))
	}

	// Add morph
	if morphSpec != "" {
		queryParts = append(queryParts, fmt.Sprintf("%s_morph=%s", ring, morphSpec))
	}

	// Join query parts with & and send without URL encoding
	queryString := ""
	for i, part := range queryParts {
		if i > 0 {
			queryString += "&"
		}
		queryString += part
	}

	_, err := c.SendRawQuery(ctx, queryString)
	return err
}

// SetLogo controls the Dynatrace logo LED
func (c *Client) SetLogo(ctx context.Context, state string) error {
	query := "logo=" + state
	_, err := c.SendRawQuery(ctx, query)
	return err
}

// SetBrightness sets the global brightness level (0-255)
func (c *Client) SetBrightness(ctx context.Context, level int) error {
	if level < 0 || level > 255 {
		return fmt.Errorf("brightness level must be between 0 and 255, got %d", level)
	}

	query := fmt.Sprintf("dim=%d", level)
	_, err := c.SendRawQuery(ctx, query)
	return err
}

// PlayEffect executes a predefined effect
func (c *Client) PlayEffect(ctx context.Context, effectQuery string) error {
	_, err := c.SendRawQuery(ctx, effectQuery)
	return err
}
