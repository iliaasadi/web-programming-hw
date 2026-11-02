FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /out/proxy .

FROM alpine:3.20
COPY --from=build /out/proxy /proxy
EXPOSE 80
ENTRYPOINT ["/proxy"]
