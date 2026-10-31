# 多阶段构建：编译在 golang 镜像，运行在 distroless（无 shell，攻击面小）。
FROM golang:{{GO_MINOR}}-alpine AS build

WORKDIR /src

# 先复制依赖清单，利用层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 静态编译，去掉调试信息，减小镜像体积
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/{{SERVICE}} ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/{{SERVICE}} /{{SERVICE}}

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/{{SERVICE}}"]
