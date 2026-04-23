package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	defaultScope  = "https://www.googleapis.com/auth/adwords"
	authEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenEndpoint = "https://oauth2.googleapis.com/token"
)

type callbackResult struct {
	Code  string
	State string
	Err   error
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func main() {
	log.SetFlags(0)

	clientID := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_CLIENT_ID"))
	if clientID == "" {
		log.Fatal("missing BE_GOOGLE_ADS_CLIENT_ID")
	}
	clientSecret := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_CLIENT_SECRET"))

	scope := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_OAUTH_SCOPE"))
	if scope == "" {
		scope = defaultScope
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	state, err := randomString(24)
	if err != nil {
		log.Fatal(err)
	}
	verifier, err := randomString(64)
	if err != nil {
		log.Fatal(err)
	}
	challenge := pkceChallenge(verifier)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen callback server: %v", err)
	}
	defer listener.Close()

	redirectURL := fmt.Sprintf("http://%s/oauth2callback", listener.Addr().String())
	callbackCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		if gotState := r.URL.Query().Get("state"); gotState != state {
			callbackCh <- callbackResult{Err: fmt.Errorf("state mismatch: got %q", gotState)}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
			desc := r.URL.Query().Get("error_description")
			callbackCh <- callbackResult{Err: fmt.Errorf("oauth error=%s description=%s", oauthErr, desc)}
			http.Error(w, "oauth authorization failed", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			callbackCh <- callbackResult{Err: errors.New("callback missing code")}
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		_, _ = io.WriteString(w, "Google OAuth authorization succeeded. You can close this tab.\n")
		callbackCh <- callbackResult{Code: code, State: state}
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := buildAuthURL(clientID, redirectURL, scope, state, challenge)
	log.Printf("Open this URL if the browser does not launch automatically:\n%s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		log.Printf("browser auto-open failed: %v", err)
	}

	var callback callbackResult
	select {
	case <-ctx.Done():
		log.Fatal("authorization cancelled")
	case callback = <-callbackCh:
	}
	if callback.Err != nil {
		log.Fatal(callback.Err)
	}

	token, err := exchangeCode(ctx, clientID, clientSecret, redirectURL, verifier, callback.Code)
	if err != nil {
		log.Fatal(err)
	}

	output, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(output))
	fmt.Println()
	fmt.Printf("export BE_GOOGLE_ADS_CLIENT_ID=%s\n", shellQuote(clientID))
	if clientSecret != "" {
		fmt.Printf("export BE_GOOGLE_ADS_CLIENT_SECRET=%s\n", shellQuote(clientSecret))
	}
	if token.RefreshToken != "" {
		fmt.Printf("export BE_GOOGLE_ADS_REFRESH_TOKEN=%s\n", shellQuote(token.RefreshToken))
	} else {
		fmt.Println("# refresh_token was empty; add prompt=consent or revoke prior grant before retrying")
	}
}

func buildAuthURL(clientID, redirectURL, scope, state, challenge string) string {
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("redirect_uri", redirectURL)
	values.Set("response_type", "code")
	values.Set("scope", scope)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	values.Set("state", state)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	return authEndpoint + "?" + values.Encode()
}

func exchangeCode(ctx context.Context, clientID, clientSecret, redirectURL, verifier, code string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange auth code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth token exchange failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, errors.New("oauth response missing access_token")
	}
	return &token, nil
}

func randomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func openBrowser(target string) error {
	cmd := exec.Command("open", target)
	return cmd.Start()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
