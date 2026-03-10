package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignIn performs email/password login and returns the session cookie string.
func SignIn(ctx context.Context, baseURL, email, password string) (string, error) {
	body, err := json.Marshal(SignInRequest{Email: email, Password: password})
	if err != nil {
		return "", fmt.Errorf("marshal sign-in: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/api/auth/sign-in"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create sign-in request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sign-in request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign-in failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// Extract cookies from Set-Cookie headers
	var cookies []string
	for _, c := range resp.Cookies() {
		cookies = append(cookies, c.Name+"="+c.Value)
	}

	if len(cookies) == 0 {
		return "", fmt.Errorf("sign-in succeeded but no cookies returned")
	}

	return strings.Join(cookies, "; "), nil
}
