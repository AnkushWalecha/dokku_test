FROM golang:1.21.0

RUN mkdir /app
WORKDIR /app

ADD . /app
RUN go install ./...

CMD /go/bin/greeter_server
