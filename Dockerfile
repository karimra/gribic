FROM golang:1.19.5 as builder
ADD . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gribic .

FROM alpine

LABEL maintainer="Karim Radhouani <medkarimrdi@gmail.com>"
LABEL documentation="https://gribic.kmrd.dev"
LABEL repo="https://github.com/karimra/gribic"
COPY --from=builder /build/gribic /app/
ENTRYPOINT [ "/app/gribic" ]
CMD [ "help" ]
