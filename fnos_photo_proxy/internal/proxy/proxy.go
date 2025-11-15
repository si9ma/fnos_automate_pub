package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"fnos_photo_proxy/internal/client"
	"fnos_photo_proxy/internal/config"
	"fnos_photo_proxy/internal/database"

	"github.com/gin-gonic/gin"
)

type ProxyServer struct {
	config     *config.Config
	httpClient *client.HTTPClient
	database   *database.Database
}

type MagicSearchRequest struct {
	Keyword string `json:"keyword"`
}

type ProxyResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func NewProxyServer(cfg *config.Config) *ProxyServer {
	db, err := database.NewDatabase(cfg.SQLiteDBPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	return &ProxyServer{
		config:     cfg,
		httpClient: client.NewHTTPClient(cfg.AutomateURL),
		database:   db,
	}
}

func (p *ProxyServer) Run(addr string) error {
	router := gin.Default()

	// 设置代理中间件
	router.Use(p.proxyMiddleware())

	return router.Run(addr)
}

func (p *ProxyServer) proxyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否是特殊的魔法搜索接口
		if c.Request.Method == "POST" && c.Request.URL.Path == "/p/api/v1/magic-search/do" {
			p.handleMagicSearch(c)
			return
		}

		// 其他请求直接代理到 fnos_base_url
		p.proxyToFnos(c)
	}
}

func (p *ProxyServer) handleMagicSearch(c *gin.Context) {
	// 读取请求body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ProxyResponse{
			Code: 1,
			Msg:  "Failed to read request body",
			Data: nil,
		})
		return
	}

	// 解析请求
	var req MagicSearchRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ProxyResponse{
			Code: 1,
			Msg:  "Invalid JSON format",
			Data: nil,
		})
		return
	}

	// 检查是否有 ocr: 前缀
	if !strings.HasPrefix(req.Keyword, "ocr:") {
		// 没有 ocr: 前缀，直接转发请求
		p.forwardRequest(c, body)
		return
	}

	// 有 ocr: 前缀，执行特殊逻辑
	ocrText := strings.TrimPrefix(req.Keyword, "ocr:")
	p.handleOCRSearch(c, ocrText)
}

func (p *ProxyServer) handleOCRSearch(c *gin.Context, ocrText string) {
	// 0. 从请求中提取 cookies
	requestCookies := c.Request.Cookies()
	cookies := make([]client.Cookie, 0)
	for _, cookie := range requestCookies {
		cookies = append(cookies, client.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   cookie.Path,
		})
	}
	// get Accesstoken header
	accesstoken := c.GetHeader("accesstoken")

	// 1. 调用 Immich API 搜索图片
	start := time.Now()
	immichResp, err := p.httpClient.SearchImmichOCR(p.config.ImmichURL, p.config.ImmichAPIKey, ocrText)
	if err != nil {
		log.Printf("Failed to search Immich OCR: %v", err)
		c.JSON(http.StatusInternalServerError, ProxyResponse{
			Code: 1,
			Msg:  "Failed to search OCR",
			Data: nil,
		})
		return
	}
	log.Printf("Immich OCR search completed in %v, found %d items", time.Since(start), len(immichResp.Assets.Items))

	// 2. 获取所有 originalPath 并应用路径替换
	filePaths := make([]string, 0)
	for _, item := range immichResp.Assets.Items {
		originalPath := item.OriginalPath
		// 应用路径替换配置
		for oldPath, newPath := range p.config.PathReplace {
			if strings.HasPrefix(originalPath, oldPath) {
				originalPath = strings.Replace(originalPath, oldPath, newPath, 1)
				break
			}
		}
		filePaths = append(filePaths, originalPath)
	}

	if len(filePaths) == 0 {
		c.JSON(http.StatusOK, ProxyResponse{
			Code: 0,
			Msg:  "success",
			Data: []interface{}{},
		})
		return
	}

	start = time.Now()
	// 3. 从数据库查询对应的 photo ID
	photoIDs, err := p.database.GetPhotoIDsByPaths(filePaths)
	if err != nil {
		log.Printf("Failed to get photo IDs from database: %v", err)
		c.JSON(http.StatusInternalServerError, ProxyResponse{
			Code: 1,
			Msg:  "Failed to query database",
			Data: nil,
		})
		return
	}
	log.Printf("Database query completed in %v, found %d photo IDs", time.Since(start), len(photoIDs))

	if len(photoIDs) == 0 {
		c.JSON(http.StatusOK, ProxyResponse{
			Code: 0,
			Msg:  "success",
			Data: []interface{}{},
		})
		return
	}

	// 如果是来自手机端请求，直接返回idList即可
	if strings.Contains(c.GetHeader("User-Agent"), "okhttp") {
		c.JSON(http.StatusOK, ProxyResponse{
			Code: 0,
			Msg:  "success",
			Data: map[string]interface{}{
				"idList": photoIDs,
			},
		})
		return
	}

	// 4. 并发调用 Fnos API 获取详情
	start = time.Now()
	galleryItems, err := p.httpClient.GetFnosGalleryItemsConcurrently(p.config.FnosBaseURL, photoIDs, cookies, accesstoken)
	if err != nil {
		log.Printf("Failed to get gallery items: %v", err)
		c.JSON(http.StatusInternalServerError, ProxyResponse{
			Code: 1,
			Msg:  "Failed to get gallery items",
			Data: nil,
		})
		return
	}
	log.Printf("Fnos gallery items fetch completed in %v", time.Since(start))

	// build idList
	idList := make([]interface{}, 0, len(galleryItems))
	for _, item := range galleryItems {
		if item, ok := item.(map[string]interface{}); ok {
			idList = append(idList, item["id"])
		}
	}

	// 5. 返回结果
	c.JSON(http.StatusOK, ProxyResponse{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"list":   galleryItems,
			"idList": idList,
		},
	})
}

func (p *ProxyServer) forwardRequest(c *gin.Context, body []byte) {
	// 创建新的请求体
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	// 代理到 fnos_base_url
	p.proxyToFnos(c)
}

func (p *ProxyServer) proxyToFnos(c *gin.Context) {
	target, err := url.Parse(p.config.FnosBaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ProxyResponse{
			Code: 1,
			Msg:  "Invalid proxy target URL",
			Data: nil,
		})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义错误处理
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte(fmt.Sprintf(`{"code": 1, "msg": "Proxy error: %v", "data": null}`, err)))
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
