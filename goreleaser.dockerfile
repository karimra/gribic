FROM alpine

LABEL maintainer="Karim Radhouani <medkarimrdi@gmail.com>"
LABEL documentation="https://gribic.kmrd.dev"
LABEL repo="https://github.com/karimra/gribic"

COPY gribic /app/gribic
ENTRYPOINT [ "/app/gribic" ]
CMD [ "help" ]
