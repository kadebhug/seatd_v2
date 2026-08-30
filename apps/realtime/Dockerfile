FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY internal ./internal
COPY apps/realtime ./apps/realtime
RUN go build -o /out/realtime ./apps/realtime/cmd/realtime

FROM alpine:3.22
COPY --from=build /out/realtime /usr/local/bin/realtime
EXPOSE 8082
ENTRYPOINT ["realtime"]
