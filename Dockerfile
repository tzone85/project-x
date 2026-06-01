FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    tmux \
    git \
    github-cli \
    tzdata \
  && adduser -D -u 1000 -h /home/px px

COPY px /usr/local/bin/px

USER px
WORKDIR /home/px

ENTRYPOINT ["/usr/local/bin/px"]
CMD ["--help"]
