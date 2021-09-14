FROM golang

ADD . /app
WORKDIR /app

RUN go build --buildmode=exe -o google-drive-mirror .

CMD ./google-drive-mirror
