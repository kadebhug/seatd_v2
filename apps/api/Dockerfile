FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY internal ./internal
COPY apps/api ./apps/api
RUN go build -o /out/api ./apps/api/cmd/api

FROM alpine:3.22
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8080
ENTRYPOINT ["api"]
