FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY internal ./internal
COPY apps/worker ./apps/worker
RUN go build -o /out/worker ./apps/worker/cmd/worker

FROM alpine:3.22
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["worker"]
