FROM golang AS builder

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.30.0

COPY ./ /go/src/github.com/zostay/dotfiles-go/
WORKDIR /go/src/github.com/zostay/dotfiles-go
RUN CGO_ENABLED=0 scripts/install.sh



FROM debian AS runner

RUN apt-get update
RUN apt-get install --yes zsh lastpass-cli

ENV HOME=/home/sterling
COPY --from=builder /go/bin/forward-file /bin/forward-file
COPY --from=builder /go/bin/label-mail /bin/label-mail
COPY ./dist/entrypoint.sh /entrypoint.sh

VOLUME /home/sterling

ENV UID=1000

ENTRYPOINT ["/entrypoint.sh"]
