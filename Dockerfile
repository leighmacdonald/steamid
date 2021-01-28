FROM golang:1.15-alpine AS backend
RUN apk add make gcc git
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN make lin

FROM alpine:latest
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
WORKDIR /app
COPY --from=backend /build/build/linux64/steamid .
ENTRYPOINT ["/app/steamid"]
CMD ["help"]