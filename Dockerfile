FROM golang

ADD . /app
WORKDIR /app

RUN go build --buildmode=exe -o drive-mirror .

CMD ./drive-mirror
