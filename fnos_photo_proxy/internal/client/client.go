package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Cookie 结构体
type Cookie struct {
	Domain   string `json:"domain"`
	HttpOnly bool   `json:"httpOnly"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	SameSite string `json:"sameSite"`
	Secure   bool   `json:"secure"`
	Value    string `json:"value"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Data []Cookie `json:"data"`
}

// SignRequest 签名请求
type SignRequest struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Data   interface{} `json:"data"`
	Params interface{} `json:"params"`
}

// SignResponse 签名响应
type SignResponse struct {
	Result string `json:"result"`
	Status string `json:"status"`
}

type ImmichSearchRequest struct {
	Page      int    `json:"page"`
	Size      int    `json:"size"`
	IsVisible bool   `json:"isVisible"`
	Language  string `json:"language"`
	OCR       string `json:"ocr"`
}

type ImmichAssetItem struct {
	OriginalPath string `json:"originalPath"`
}

type ImmichAssets struct {
	Total int               `json:"total"`
	Count int               `json:"count"`
	Items []ImmichAssetItem `json:"items"`
}

type ImmichSearchResponse struct {
	Assets ImmichAssets `json:"assets"`
}

type FnosGalleryResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type HTTPClient struct {
	client      *http.Client
	automateURL string
}

func NewHTTPClient(automateURL string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		automateURL: automateURL,
	}
}

// GetLoginCookie 获取登录cookie
func (c *HTTPClient) GetLoginCookie() ([]Cookie, error) {
	url := fmt.Sprintf("%s/login", c.automateURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get login cookie: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var loginResp LoginResponse
	err = json.Unmarshal(body, &loginResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	return loginResp.Data, nil
}

// GetSign 获取签名
func (c *HTTPClient) GetSign(method, url string, data, params interface{}) (string, error) {
	signReq := SignRequest{
		Method: method,
		URL:    url,
		Data:   data,
		Params: params,
	}

	jsonData, err := json.Marshal(signReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sign request: %w", err)
	}

	signURL := fmt.Sprintf("%s/gen_photo_sign", c.automateURL)
	req, err := http.NewRequest("POST", signURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create sign request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute sign request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sign request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read sign response body: %w", err)
	}

	var signResp SignResponse
	err = json.Unmarshal(body, &signResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal sign response: %w", err)
	}

	if signResp.Status != "success" {
		return "", fmt.Errorf("sign request failed with status: %s", signResp.Status)
	}

	return signResp.Result, nil
}

func (c *HTTPClient) SearchImmichOCR(immichURL, apiKey, ocrText string) (*ImmichSearchResponse, error) {
	req := ImmichSearchRequest{
		Page:      1,
		Size:      1000,
		IsVisible: true,
		Language:  "en",
		OCR:       ocrText,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/search/metadata?apiKey=%s", immichURL, apiKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("immich API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result ImmichSearchResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (c *HTTPClient) GetFnosGalleryItem(fnosURL string, id int64, cookies []Cookie, accesstoken string) (*FnosGalleryResponse, error) {
	// 构建API路径
	path := fmt.Sprintf("/p/api/v1/gallery/getOne?id=%d", id)
	url := fmt.Sprintf("%s%s", fnosURL, path)

	// 获取签名
	sign, err := c.GetSign("GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置cookies
	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   cookie.Path,
		})
	}

	// 设置认证头
	req.Header.Set("authx", sign)
	req.Header.Set("Accesstoken", accesstoken)

	// 执行请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get fnos gallery item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gallery request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result FnosGalleryResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (c *HTTPClient) GetFnosGalleryItemsConcurrently(fnosURL string, ids []int64, cookies []Cookie, accesstoken string) ([]interface{}, error) {
	if len(ids) == 0 {
		return []interface{}{}, nil
	}

	// 使用 channel 来收集结果和错误
	type result struct {
		index int
		data  interface{}
		err   error
	}

	resultChan := make(chan result, len(ids))

	// 启动所有 goroutine
	for i, id := range ids {
		go func(index int, photoID int64) {
			resp, err := c.GetFnosGalleryItem(fnosURL, photoID, cookies, accesstoken)
			if err != nil {
				resultChan <- result{
					index: index,
					data:  nil,
					err:   fmt.Errorf("failed to get gallery item %d: %w", photoID, err),
				}
				return
			}

			resultChan <- result{
				index: index,
				data:  resp.Data,
				err:   nil,
			}
		}(i, id)
	}

	// 收集所有结果
	results := make([]interface{}, len(ids))
	errors := make([]error, 0)

	for i := 0; i < len(ids); i++ {
		res := <-resultChan
		if res.err != nil {
			errors = append(errors, res.err)
		} else {
			results[res.index] = res.data
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("errors occurred while fetching gallery items: %v", errors)
	}

	// 过滤掉 nil 值
	filteredResults := make([]interface{}, 0)
	for _, result := range results {
		if result != nil {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults, nil
}
