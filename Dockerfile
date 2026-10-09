FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 appuser
WORKDIR /app
COPY boureminder-api .
USER appuser
EXPOSE 8080
CMD ["./boureminder-api"]
