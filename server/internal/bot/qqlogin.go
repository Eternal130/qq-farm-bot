package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	chromeUA  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	qua       = "V1_HT5_QDT_0.70.2209190_x64_0_DEV_D"
	farmAppID = "1112386029"
)

type QRLoginResult struct {
	QRCodeURL string `json:"qr_code_url"`
	LoginCode string `json:"login_code"`
}

type QRLoginStatus struct {
	Status  string `json:"status"`            // "wait", "ok", "expired", "error"
	Code    string `json:"code,omitempty"`    // auth code on success
	Message string `json:"message,omitempty"` // error detail for frontend display
}

func qqHeaders() http.Header {
	h := http.Header{}
	h.Set("qua", qua)
	h.Set("host", "q.qq.com")
	h.Set("accept", "application/json")
	h.Set("content-type", "application/json")
	h.Set("user-agent", chromeUA)
	return h
}

// RequestQRCode initiates a QQ scan login and returns the QR URL.
func RequestQRCode() (*QRLoginResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", "https://q.qq.com/ide/devtoolAuth/GetLoginCode", nil)
	req.Header = qqHeaders()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Code != 0 || result.Data.Code == "" {
		return nil, fmt.Errorf("获取扫码登录码失败")
	}

	return &QRLoginResult{
		LoginCode: result.Data.Code,
		QRCodeURL: fmt.Sprintf("https://h5.qzone.qq.com/qqq/code/%s?_proxy=1&from=ide", result.Data.Code),
	}, nil
}

// PollQRStatus checks the scan status.
// Returns a status object with NO error for all expected QR states (wait/ok/expired),
// so the API handler always returns HTTP 200 and the frontend can react properly.
// Only returns a Go error for truly unexpected failures (network, JSON parse).
func PollQRStatus(loginCode string) (*QRLoginStatus, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	pollURL := fmt.Sprintf(
		"https://q.qq.com/ide/devtoolAuth/syncScanSateGetTicket?code=%s",
		url.QueryEscape(loginCode),
	)
	req, _ := http.NewRequest("GET", pollURL, nil)
	req.Header = qqHeaders()
	resp, err := client.Do(req)
	if err != nil {
		return &QRLoginStatus{Status: "error", Message: "网络请求失败"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &QRLoginStatus{Status: "error", Message: fmt.Sprintf("QQ服务器返回 %d", resp.StatusCode)}, nil
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Ok     int    `json:"ok"`
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &QRLoginStatus{Status: "error", Message: "解析响应失败"}, nil
	}

	if result.Code == -10003 {
		return &QRLoginStatus{Status: "expired"}, nil
	}
	if result.Code != 0 {
		return &QRLoginStatus{Status: "error", Message: fmt.Sprintf("QQ返回错误码 %d", result.Code)}, nil
	}
	if result.Data.Ok != 1 {
		return &QRLoginStatus{Status: "wait"}, nil
	}

	// User scanned — exchange ticket for auth code
	authCode, err := getAuthCode(result.Data.Ticket)
	if err != nil {
		return &QRLoginStatus{Status: "error", Message: err.Error()}, nil
	}
	return &QRLoginStatus{Status: "ok", Code: authCode}, nil
}

// getAuthCode exchanges a scan ticket for a farm login code.
// Handles both string and numeric "code" in the QQ API response,
// matching Node.js behavior which uses implicit type coercion.
func getAuthCode(ticket string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	payload, _ := json.Marshal(map[string]string{"appid": farmAppID, "ticket": ticket})

	req, _ := http.NewRequest("POST", "https://q.qq.com/ide/login", bytes.NewReader(payload))
	req.Header = qqHeaders()

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求登录接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("登录接口返回 HTTP %d", resp.StatusCode)
	}

	// Read body once, try flexible parsing (QQ API may return code as string or number)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// Try 1: {"code": "string_value"}
	var strResult struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &strResult); err == nil && strResult.Code != "" {
		return strResult.Code, nil
	}

	// Try 2: {"code": 12345} (numeric code, Node.js handles via implicit coercion)
	var numResult struct {
		Code json.Number `json:"code"`
	}
	if err := json.Unmarshal(body, &numResult); err == nil && numResult.Code.String() != "" && numResult.Code.String() != "0" {
		return numResult.Code.String(), nil
	}

	return "", fmt.Errorf("获取农场登录 code 失败 (响应: %s)", string(body))
}
