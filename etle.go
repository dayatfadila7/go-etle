package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ETLEClient struct {
	http *http.Client
}

func NewETLEClient() *ETLEClient {
	return &ETLEClient{http: &http.Client{Timeout: 30 * time.Second}}
}

func (e *ETLEClient) baseURL(c *Client) string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return "https://api-etle.polri.go.id"
}

// Login ke /user/login, kembalikan access_token + refreshToken.
func (e *ETLEClient) Login(c *Client) (*LoginResponse, error) {
	payload := map[string]string{
		"usertoken":     c.Usertoken,
		"passtoken":     c.Passtoken,
		"client_secret": c.ClientSecret,
		"client_id":     c.ClientID,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", e.baseURL(c)+"/user/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &APIError{resp.StatusCode, string(body)}
	}
	var lr LoginResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, fmt.Errorf("decode login: %w", err)
	}
	return &lr, nil
}

// IsTokenExpired menandai error HTTP ETLE yang berarti JWT sudah kedaluwarsa.
// ETLE mengembalikan 401 kadang, tetapi untuk JWT expired seringnya 403
// {"name":"TokenExpiredError","message":"jwt expired",...}.
func IsTokenExpired(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	if apiErr.StatusCode != 401 && apiErr.StatusCode != 403 {
		return false
	}
	body := strings.ToLower(apiErr.Body)
	return strings.Contains(body, "jwt expired") ||
		strings.Contains(body, "token expired") ||
		strings.Contains(body, "tokenexpirederror") ||
		strings.Contains(body, "\"expired\"")
}

// TokenExpired mengecek masa berlaku JWT access_token secara lokal (decode exp),
// tanpa memanggil server. Token kosong dianggap expired agar dipaksa login ulang.
func TokenExpired(token string) bool {
	if token == "" {
		return true
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	return claims.Exp > 0 && time.Now().Unix() >= claims.Exp
}

func (e *ETLEClient) SyncMaster(c *Client) ([]MasterViolation, error) {
	tok := c.AuthToken
	if tok == "" {
		tok = c.Usertoken // ponytail: fallback sebelum login
	}
	req, _ := http.NewRequest("GET", e.baseURL(c)+"/master/list", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		apiErr := &APIError{resp.StatusCode, string(body)}
		if !IsTokenExpired(apiErr) {
			return nil, apiErr
		}
		// token kedaluwarsa -> login ulang sekali lalu coba lagi
		lr, lerr := e.Login(c)
		if lerr != nil {
			return nil, apiErr
		}
		c.AuthToken = lr.AccessToken
		c.RefreshToken = lr.RefreshToken
		req2, _ := http.NewRequest("GET", e.baseURL(c)+"/master/list", nil)
		req2.Header.Set("Authorization", "Bearer "+lr.AccessToken)
		req2.Header.Set("Accept", "application/json")
		resp2, err2 := e.http.Do(req2)
		if err2 != nil {
			return nil, err2
		}
		defer resp2.Body.Close()
		body, _ = io.ReadAll(resp2.Body)
		if resp2.StatusCode != 200 {
			return nil, &APIError{resp2.StatusCode, string(body)}
		}
	}
	return parseMaster(body)
}

// ponytail: response /master/list belum pasti JSON; kalau XML perlu decode terpisah
func parseMaster(body []byte) ([]MasterViolation, error) {
	var any interface{}
	if err := json.Unmarshal(body, &any); err != nil {
		return nil, err
	}
	out := []MasterViolation{}
	seen := map[string]bool{}
	findMaster(any, &out, seen)
	return out, nil
}

func findMaster(v interface{}, out *[]MasterViolation, seen map[string]bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		if code, name := extractKV(t); code != "" && name != "" && !seen[code] {
			seen[code] = true
			*out = append(*out, MasterViolation{Code: code, Name: name})
		}
		for _, val := range t {
			findMaster(val, out, seen)
		}
	case []interface{}:
		for _, val := range t {
			findMaster(val, out, seen)
		}
	}
}

func extractKV(m map[string]interface{}) (string, string) {
	codeKeys := []string{"code", "kode", "violation_code", "violationCode", "id"}
	nameKeys := []string{"name", "nama", "violation_name", "violationName", "description", "desc"}
	code, name := "", ""
	for _, k := range codeKeys {
		if v, ok := m[k]; ok {
			code = toStr(v)
			break
		}
	}
	for _, k := range nameKeys {
		if v, ok := m[k]; ok {
			name = toStr(v)
			break
		}
	}
	return code, name
}

func toStr(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// FetchPublicMaster ambil master pelanggaran dari /master/list (endpoint publik, tanpa auth).
func (e *ETLEClient) FetchPublicMaster() ([]MasterViolation, error) {
	req, _ := http.NewRequest("GET", "https://api-etle.polri.go.id/master/list", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &APIError{resp.StatusCode, string(body)}
	}
	return parseMaster(body)
}

func (e *ETLEClient) SendViolation(c *Client, p ViolationPayload) (*SendResponse, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", e.baseURL(c)+"/violation/insert", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &APIError{resp.StatusCode, string(body)}
	}
	var out SendResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode: %w body=%s", err, string(body))
	}
	return &out, nil
}
