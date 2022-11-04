FROM golang:1.19-alpine AS builder

RUN apk add build-base

RUN mkdir /build

ADD go.mod /build/go.mod
ADD go.sum /build/go.sum

ADD ./cmd /build/cmd

WORKDIR /build

RUN go mod download


RUN GOOS=linux go build -tags musl -a --mod=readonly -installsuffix cgo -ldflags "-X 'main.buildtime=$(date -u '+%Y-%m-%d %H:%M:%S')' -extldflags '-static'" -o mainFile ./cmd

FROM alpine AS runner
COPY --from=builder /build /app/
WORKDIR /app
CMD ["./mainFile"]