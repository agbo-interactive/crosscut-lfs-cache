FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/lfscache .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /out/lfscache /bin/lfscache
ENTRYPOINT ["/bin/lfscache"]
