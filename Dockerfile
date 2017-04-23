FROM golang:alpine

COPY . /go/src/app

WORKDIR /go/src/app

RUN go-wrapper download
RUN go-wrapper install

# To be able to use netem to simulate network problems
RUN apk add --update iproute2

CMD ["go-wrapper", "run", "-stderrthreshold", "0"]
