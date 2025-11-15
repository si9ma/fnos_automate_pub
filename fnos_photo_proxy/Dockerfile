FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache build-base

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

# 安装 sqlite
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /root/

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /app/main .

# 复制配置文件
COPY config.json .

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]