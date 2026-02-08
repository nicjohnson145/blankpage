FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o blankpage-server ./cmd/server 

FROM alpine:3.23.3
COPY --from=builder /src/blankpage-server /bin/blankpage-server
ENTRYPOINT ["/bin/blankpage-server"]
