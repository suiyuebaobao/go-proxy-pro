# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git make

# 复制 go mod 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建前端
WORKDIR /app/web
RUN npm install
RUN npm run build

# 构建后端
WORKDIR /app
RUN make build

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 复制二进制文件和前端资源
COPY --from=builder /app/aiproxy /app/
COPY --from=builder /app/internal/handler/dist /app/internal/handler/dist/
COPY --from=builder /app/configs /app/configs/

# 创建日志目录
RUN mkdir -p logs

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 运行应用
CMD ["./aiproxy", "server"]
