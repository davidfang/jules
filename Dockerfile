# Dockerfile for go-base-system-v2

# ---- Build Stage ----
# 使用官方的 Golang Alpine 镜像作为构建环境。
# Alpine 镜像是轻量级的，有助于减小最终镜像的大小。
# 指定 Go 版本以确保构建环境的一致性。
FROM golang:1.21-alpine AS builder

# 设置工作目录，后续的命令将在此目录下执行。
WORKDIR /app

# 复制 Go 模块依赖文件。
# 单独复制这些文件并执行 go mod download 可以利用 Docker 的层缓存机制。
# 只有当 go.mod 或 go.sum 文件发生变化时，才会重新下载依赖。
COPY go.mod go.sum ./
RUN go mod download

# 复制项目源码到工作目录。
COPY . .

# 编译 Go 应用。
# CGO_ENABLED=0: 禁用 CGO，以构建静态链接的二进制文件，避免对 C 库的依赖。
# GOOS=linux: 指定目标操作系统为 Linux (与 Alpine 基础镜像兼容)。
# -ldflags="-s -w": 编译选项，用于减小二进制文件的大小。
#   -s: 忽略符号表。
#   -w: 忽略 DWARF 调试信息。
# -o /app/main: 指定输出的二进制文件名为 main，并放在 /app 目录下 (在 builder 阶段)。
# 最后的 "." 表示在当前工作目录 (即 /app) 下查找 main 包进行编译。
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main .

# ---- Runtime Stage ----
# 使用轻量级的 Alpine 镜像作为最终的运行时环境。
FROM alpine:latest

# 创建一个非 root 用户和组，以增强安全性。
# 不以 root 用户运行应用是一种最佳实践。
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 将 Build Stage 中编译好的二进制文件复制到 Runtime Stage。
# 二进制文件放在新用户的主目录下。
COPY --from=builder /app/main /home/appuser/main

# 复制应用的配置文件目录到新用户的主目录下。
# config 目录通常包含 config.yaml 和 casbin_model.conf 等。
COPY ./config /home/appuser/config

# 设置工作目录为新用户的主目录。
WORKDIR /home/appuser

# 更改二进制文件和配置文件目录的所有权给新创建的 appuser。
# 确保应用以非 root 用户运行时有权限访问这些文件。
RUN chown -R appuser:appgroup /home/appuser/main /home/appuser/config

# 切换到非 root 用户 appuser 来运行应用。
USER appuser

# 暴露应用监听的端口。
# 这个端口应与 config.yaml 中 server.port 以及 docker-compose.yml 中映射的端口一致。
# Dockerfile 中的 EXPOSE 仅为文档作用，实际端口映射在 docker-compose.yml 或 docker run 命令中完成。
EXPOSE 8080

# 设置容器启动时执行的命令。
# ["./main", "server"] 表示执行当前工作目录下的 main 二进制文件，并传递 "server" 作为参数。
# 这会启动 Cobra 应用的 server 子命令。
ENTRYPOINT ["./main", "server"]

# 可以添加 CMD 来提供 ENTRYPOINT 的默认参数，但在这里 ENTRYPOINT 已经包含了完整的命令。
# CMD ["server"] # 如果 ENTRYPOINT 是 ["./main"]，则可以使用此 CMD
