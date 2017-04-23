FROM golang:alpine

COPY . /go/src/app

WORKDIR /go/src/app

RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run", "-stderrthreshold", "1"]
