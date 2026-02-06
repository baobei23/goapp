FROM golang:1.25.5 AS builder

COPY . /app
WORKDIR /app
# Toggle CGO on your app requirement
RUN CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o /app/appbin *.go

FROM alpine:3.23
LABEL maintainer="Author <reginaldsaja98@gmail.com>"


# Install CA certs and dependencies for Alpine
RUN apk update && apk add --no-cache \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Add new user 'appuser'
RUN adduser -D -h /home/appuser appuser
USER appuser

COPY --from=builder /app/cmd/server/http/web /home/appuser/app/web
COPY --from=builder /app/appbin /home/appuser/app/appbin

ENV TEMPLATES_BASEPATH=/home/appuser/app/web/templates

WORKDIR /home/appuser/app

# Expose ports for HTTP, Health, and Metrics
EXPOSE 8080 2000 9090

CMD ["./appbin"]