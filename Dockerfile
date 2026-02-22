FROM oven/bun:1 AS ui_base
WORKDIR /usr/src/app

COPY ui /usr/src/app/
RUN bun install --frozen-lockfile
ENV NODE_ENV=production
RUN bun run build

FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
COPY --from=ui_base /usr/src/app/dist ./cmd/server/dist
RUN CGO_ENABLED=0 go build -o blankpage-server ./cmd/server 

FROM alpine:3.23.3
COPY --from=builder /src/blankpage-server /bin/blankpage-server
ENTRYPOINT ["/bin/blankpage-server"]
