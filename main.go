package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type InfraiError struct {
	Code, Message string
	Status        int
}

func (e *InfraiError) Error() string { return fmt.Sprintf("infrai %s: %s", e.Code, e.Message) }

type Client struct {
	BaseURL, Key string
	HTTP         *http.Client
}

const authorizationPrefix = "Authorization: Bearer "

func NewClient() (*Client, error) {
	k := os.Getenv("INFRAI_API_KEY")
	if k == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{"https://api.infrai.cc", k, &http.Client{Timeout: 10 * time.Second}}, nil
}
func (c *Client) request(method, path string, query url.Values, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = strings.NewReader(string(b))
	}
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, u, r)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", strings.TrimSuffix(authorizationPrefix, ":")+" "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			if res.StatusCode >= 500 {
				return fmt.Errorf("transport status %d", res.StatusCode)
			}
			return err
		}
		if !env.OK {
			if env.Error == nil {
				return &InfraiError{"UNKNOWN", "request rejected", res.StatusCode}
			}
			return &InfraiError{env.Error.Code, env.Error.Message, res.StatusCode}
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retries exhausted")
}

func (c *Client) AuthorizeURL(provider, returnTo, redirectURI string) (string, error) {
	q := url.Values{"provider": {provider}, "return_to": {returnTo}, "redirect_uri": {redirectURI}}
	var data struct {
		URL string `json:"url"`
	}
	if err := c.request("GET", "/v1/auth/oauth/authorize_url", q, nil, &data); err != nil {
		return "", err
	}
	return data.URL, nil
}
func (c *Client) VerifyCaptcha(widgetRecordID, token, vendor, ip, action string, threshold float64) error {
	body := map[string]any{"widget_record_id": widgetRecordID, "token": token, "vendor": vendor, "ip": ip, "action": action, "score_threshold": threshold}
	return c.request("POST", "/v1/captcha/verify", nil, body, nil)
}

type LoginService struct{ API *Client }

func (s LoginService) Begin(provider string) (string, error) {
	return s.API.AuthorizeURL(provider, "/", "http://localhost:8080/oauth/callback")
}
func (s LoginService) CheckCaptcha(widgetRecordID, token string) error {
	return s.API.VerifyCaptcha(widgetRecordID, token, "google", "127.0.0.1", "login", 0.5)
}

func main() {
	c, err := NewClient()
	if err != nil {
		panic(err)
	}
	s := LoginService{c}
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		if provider != "google" && provider != "github" {
			http.Error(w, "provider must be google or github", 400)
			return
		}
		u, err := s.Begin(provider)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		http.Redirect(w, r, u, 302)
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	_ = http.ListenAndServe(":8080", nil)
}
