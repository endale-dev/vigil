FROM --platform=$BUILDPLATFORM node:24.22.3-bookworm-slim AS frontend-build

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run typecheck && npm run build

FROM --platform=$BUILDPLATFORM golang:1.26.11-bookworm AS backend-build

WORKDIR /src/backend
COPY backend/main.go backend/embed.go ./
COPY backend/internal/ ./internal/
COPY --from=frontend-build /src/frontend/dist/ ./dist/
RUN go mod init github.com/endale-dev/vigil \
    && go mod edit -go=1.26 \
    && go get go.mongodb.org/mongo-driver/v2@v2.10.1 \
    && go mod tidy

ARG TARGETOS
ARG TARGETARCH
RUN mkdir -p /out \
    && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
        go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/vigil .

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /
COPY --from=backend-build /out/vigil /vigil
ENV APP_ENV=production
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/vigil"]
