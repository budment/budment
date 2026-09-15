FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/budment /usr/local/bin/budment

ENTRYPOINT ["budment"]