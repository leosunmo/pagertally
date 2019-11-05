FROM alpine:3.9

COPY pagertally /pagertally

RUN apk --no-cache add tzdata ca-certificates

ENTRYPOINT ["/pagertally"]
