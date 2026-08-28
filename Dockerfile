FROM bufbuild/buf AS buf
FROM dart:stable AS sass

COPY --from=buf /usr/local/bin/buf /usr/local/bin/

RUN git clone https://github.com/sass/dart-sass.git /dart-sass && \
    cd /dart-sass && \
    dart pub get && \
    dart run grinder protobuf && \
    dart compile exe bin/sass.dart


FROM golangci/golangci-lint:latest AS golangci-lint
FROM golang:latest AS jigyll
ARG VERSION=develop

ADD . /jigyll

COPY --from=golangci-lint /usr/bin/golangci-lint /usr/bin/golangci-lint
COPY --from=sass /dart-sass/bin/sass.exe /usr/bin/sass

WORKDIR /jigyll

RUN go test ./...
RUN golangci-lint run
RUN go build -ldflags "-s -w -X github.com/reidransom/jigyll/version.Version=${VERSION}" -o /usr/bin/jigyll .

FROM debian:stable-slim

COPY --from=jigyll /usr/bin/jigyll /usr/bin/jigyll
COPY --from=sass /dart-sass/bin/sass.exe /usr/bin/sass

WORKDIR /app

ENTRYPOINT [ "/usr/bin/jigyll" ]

CMD [ "--help" ]
