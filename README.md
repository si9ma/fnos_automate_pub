# FNOS 自动化与图片代理服务

这是一个用于对 FNOS 平台请求进行代理并提供图片 OCR / 搜索能力的组合服务仓库。它包含两个主要组件：

- `fnos_selenium`：基于 Flask 的 Selenium 自动化服务，用于通过浏览器自动登录 FNOS、生成或透传签名（cookies）等自动化操作。
- `fnos_photo_proxy`：用 Go 编写的 HTTP 代理服务，拦截并代理 FNOS 的请求，同时对带有 OCR 前缀的搜索请求调用 Immich / SQLite 数据库进行图片匹配与详情拼装。

本 README 用中文说明如何配置、运行和使用本项目。

## 目录结构（相关）

```
docker-compose.yml
fnos_photo_proxy/
  ├─ main.go
  ├─ config.json (示例放在容器外部挂载)
  └─ internal/...
fnos_selenium/
  ├─ app.py
  ├─ selenium_service.py
  └─ requirements.txt
```

## 快速概览

- `fnos_selenium` 提供以下 HTTP 端点（默认运行在 5000 端口）：
  - `GET /health` - 健康检查，返回浏览器与登录状态。
  - `GET /login` - 触发 Selenium 登录（读取环境变量 `FNOS_LOGIN_URL`、`FNOS_USERNAME`、`FNOS_PASSWORD`）。
  - `POST /gen_photo_sign` - 接收参数并执行生成签名的脚本，返回执行结果。
  - `GET /info` - 返回当前浏览器页面信息。
  - `POST /shutdown` - 关闭浏览器会话。

- `fnos_photo_proxy` 提供代理功能并监听 5586（容器内对外映射为 5586）。它读取 `config.json` 来配置 FNOS、Immich、SQLite 路径、automate（Selenium）服务地址等。

## 配置

在 `fnos_photo_proxy` 的根目录有一个 `config.json` 示例（在仓库内有 README 片段），关键字段如下：

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

- port：Go 服务监听端口（内部端口，容器通常映射为 5586）。
- fnos_base_url / fnos_url：FNOS 服务的基地址，用于一般代理与获取图片详情。
- immich_url / immich_api_key：调用 Immich API 做图片关键字搜索时使用。
- automate_url：Selenium 自动化服务地址（默认 `http://localhost:5000`，若使用 docker-compose，请设置为容器名与端口，例如 `http://fnos_selenium:5000`）。
- path_replace：可选，将 Immich 返回的文件路径替换为宿主机或代理可访问的路径。
- sqlite_db_path：SQLite 数据库文件路径（照片索引），服务会读取其中 `photo` 表。

注意：仓库的 `docker-compose.yml` 将 `fnos_photo_proxy` 的 `config.json` 以卷方式挂载至容器内的 `/root/config.json`，并映射宿主的 `photo.db`、`photo.db-shm`、`photo.db-wal`（如果存在）。

## 环境变量（主要用于 `fnos_selenium`）

- FNOS_LOGIN_URL - FNOS 登录页面 URL（例如 `http://fnos.lan:5666/p`）。
- FNOS_USERNAME - 登录用户名。
- FNOS_PASSWORD - 登录密码。

这些可以通过 Docker Compose 的 `environment` 注入，或在运行容器之前在宿主机上导出。

## 运行

推荐使用 Docker Compose 一键启动（仓库根含 `docker-compose.yml`）：

```bash
docker-compose up -d
```

这会构建并启动两个服务：

- `fnos_selenium`：Flask + Selenium 服务，负责自动登录与签名生成。
- `fnos_photo_proxy`：Go 代理服务，监听宿主机 5586 端口（映射到容器内 5586）。

本地开发（不使用 Docker）

fnos_photo_proxy（Go）

```bash
cd fnos_photo_proxy
go mod tidy
go run main.go
```

fnos_selenium（Python）

```bash
cd fnos_selenium
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python app.py
```

注意：Selenium 服务需要可用的浏览器驱动（chromedriver/geckodriver 等），`selenium_service.py` 中会负责启动浏览器，确保宿主或容器内已经安装相应的浏览器与驱动。

## API 与使用示例

fnos_photo_proxy 会将除 `/p/api/v1/magic-search/do` 外的请求直接代理到 `fnos_base_url`（行为依配置而定）。对于 `/p/api/v1/magic-search/do` 的 POST 请求，如果 `keyword` 以 `ocr:` 开头，服务会触发 OCR 搜索流程：

1. 使用 Immich API（或本地策略）根据 OCR 文本搜索可能的图片，或从 SQLite（`photo` 表）中查找匹配文件路径。
2. 使用 `fnos_url`（通过代理）获取图片详情并合并信息。
3. 返回统一的 JSON 结果给调用方。

示例请求（OCR）：

POST /p/api/v1/magic-search/do

```json
{
  "keyword": "ocr:在照片中出现的文字"
}
```

示例成功响应：

```json
{
  "code": 0,
  "msg": "success",
  "data": [
    /* 照片详情数组 */
  ]
}
```

如果 `keyword` 不以 `ocr:` 开头，请求将被直接代理到 `fnos_base_url` 的对应路径。

## 注意事项与常见问题

- SQLite WAL 文件：仓库示例将 `photo.db`、`photo.db-shm`、`photo.db-wal` 一起挂载以支持 WAL 模式读取；请确保挂载的文件与数据库版本匹配。
- automate_url 配置：在 Docker Compose 下，`fnos_photo_proxy` 需要将 `automate_url` 指向 `fnos_selenium` 服务（例如 `http://fnos_selenium:5000`），否则代理无法通过 Selenium 服务生成签名或进行登录动作。
- 超时：HTTP 请求有超时限制（仓库代码注释里提到 30 秒），对外请求较慢时请适当调整配置或增加重试逻辑。
- Cookie 与认证：代理会优先使用客户端传来的 cookies；当需要时会调用 `automate_url` 让 Selenium 生成或刷新登录凭证。
- 日志：服务会将错误与重要事件记录到标准输出，建议在生产环境使用容器日志或集成日志系统收集。

## 开发与调试建议

- 调试 `fnos_selenium`：访问 `http://localhost:5000/health` 查看浏览器与登录状态；使用 `/login` 手动触发登录过程并查看返回值。
- 调试 `fnos_photo_proxy`：直接向 `http://localhost:5586/p/api/v1/magic-search/do` 发送请求测试不同关键字（普通关键字 vs `ocr:` 前缀）。

## 许可证

本项目包含两个组件各自的 LICENSE 文件，请在分发或修改前阅读相应许可证条款。

## 联系与贡献

欢迎提交 issue 或 pull request。如需帮助，请在 issue 中提供复现步骤、配置文件（去敏感信息）与日志片段。

---

生成说明：此 README 基于仓库中已有的 `docker-compose.yml`、`fnos_selenium/app.py` 与 `fnos_photo_proxy` 的 README 片段整合而成，覆盖典型的配置与运行步骤。
