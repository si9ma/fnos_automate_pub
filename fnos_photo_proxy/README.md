# FNOS Photo Proxy

一个用于代理 FNOS 请求并提供 OCR 图片搜索功能的 HTTP 代理服务。

## 功能特性

- **通用代理**: 除了特定接口外，所有请求都会代理到配置的 FNOS 基础 URL
- **Cookie 透传**: 自动从客户端请求中提取 cookies 用于 FNOS API 认证
- **自动签名**: 使用 selenium 服务获取请求签名
- **OCR 搜索**: 对于带有 `ocr:` 前缀的搜索请求，会：
  1. 调用 Immich API 搜索包含指定文本的图片
  2. 从 SQLite 数据库查询对应的照片 ID
  3. 并发调用 FNOS API 获取详细信息（使用客户端的 cookies 和自动签名）
  4. 返回组装后的结果

## 配置

编辑 `config.json` 文件：

```json
{
  "port": "8080",
  "fnos_base_url": "http://your-fnos-server.com",
  "fnos_url": "http://your-fnos-server.com", 
  "immich_url": "http://immich.lan",
  "immich_api_key": "your-immich-api-key",
  "automate_url": "http://localhost:5000",
  "path_replace": {
    "/usr/src/app/external/MobileBackup/iPhone": "/path/to/actual/photos"
  },
  "sqlite_db_path": "./photo.db"
}
```

### 配置说明

- `port`: 代理服务监听端口
- `fnos_base_url`: FNOS 服务的基础 URL（用于一般代理）
- `fnos_url`: FNOS 服务的 URL（用于获取照片详情）
- `immich_url`: Immich 服务的 URL
- `immich_api_key`: Immich API 密钥
- `automate_url`: Selenium 自动化服务的 URL（用于获取 FNOS 登录凭证和签名）
- `path_replace`: 路径替换配置，用于将 Immich 返回的路径转换为实际的文件路径
- `sqlite_db_path`: SQLite 数据库文件路径

## 运行方式

### 直接运行

```bash
go mod tidy
go run main.go
```

### Docker 运行

```bash
docker-compose up -d
```

## API 使用

### 一般请求

所有非 `/p/api/v1/magic-search/do` 的请求会直接代理到 `fnos_base_url`。

### OCR 搜索

对 `/p/api/v1/magic-search/do` 发送 POST 请求：

```json
{
  "keyword": "ocr:搜索文本"
}
```

如果 keyword 以 `ocr:` 开头，会触发 OCR 搜索逻辑；否则直接转发到 FNOS 服务。

### 响应格式

```json
{
  "code": 0,
  "msg": "success",
  "data": [
    // 照片详情列表
  ]
}
```

## 数据库要求

服务需要访问包含 `photo` 表的 SQLite 数据库，表结构至少需要包含：

- `id`: 照片 ID
- `file_path`: 文件路径

## 注意事项

- 服务会并发访问 SQLite 数据库，使用了适当的连接管理
- 支持 SQLite 的 WAL 模式，不会影响其他进程的读写操作
- 所有 HTTP 请求都有 30 秒超时限制
- 错误会被适当记录到日志中
- **认证方式**: 
  - Cookies 从客户端请求中自动提取并透传给 FNOS API
  - 请求签名通过 `automate_url` 配置的 selenium 服务获取
- **依赖服务**: 需要 `fnos_selenium` 服务运行在 `automate_url` 配置的地址上，用于提供 FNOS 请求签名