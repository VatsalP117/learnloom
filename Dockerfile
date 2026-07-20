# syntax=docker/dockerfile:1.7
FROM node:24-alpine AS web
WORKDIR /src
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web ./web
ARG VITE_CLERK_PUBLISHABLE_KEY
ARG VITE_LEARNLOOM_ROOT_DOMAIN=learnloom.blog
ENV VITE_CLERK_PUBLISHABLE_KEY=$VITE_CLERK_PUBLISHABLE_KEY
ENV VITE_LEARNLOOM_ROOT_DOMAIN=$VITE_LEARNLOOM_ROOT_DOMAIN
RUN npm run build

FROM golang:1.25.12-alpine AS service
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /learnloom ./cmd/learnloom

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=service --chown=nonroot:nonroot /learnloom /learnloom
COPY --from=web --chown=nonroot:nonroot /src/web/dist /app/web/dist
USER nonroot:nonroot
ENV FRONTEND_DIR=/app/web/dist
EXPOSE 3000 9090
ENTRYPOINT ["/learnloom"]
CMD ["web"]
