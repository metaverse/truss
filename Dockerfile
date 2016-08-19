# Base Docker Image
FROM alpine:3.4

RUN apk add --no-cache git
RUN apk add --no-cache go
RUN apk add --no-cache openssh
RUN apk add --update --no-cache build-base autoconf automake libtool libstdc++ curl

RUN git clone https://github.com/google/protobuf -b 3.0.0-beta-4 --depth 1
WORKDIR "/protobuf"
RUN ls -la .
RUN	./autogen.sh && \
	./configure --prefix=/usr && \
	make && \
	make install

WORKDIR "/"
RUN rm -rf protobuf

ENV GOROOT /usr/lib/go
ENV GOPATH /gopath
ENV GOBIN /gopath/bin
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

RUN go get -u -v github.com/lightstep/lightstep-tracer-go
RUN go get -u -v github.com/opentracing/opentracing-go
RUN go get -u -v github.com/openzipkin/zipkin-go-opentracing
RUN go get -u -v github.com/prometheus/client_golang/prometheus
RUN go get -u -v github.com/sourcegraph/appdash/opentracing
RUN go get -u -v golang.org/x/net/context
RUN go get -u -v sourcegraph.com/sourcegraph/appdash

RUN go get -u -v github.com/golang/protobuf/proto 
RUN go get -u -v github.com/golang/protobuf/protoc-gen-go 
RUN go get -u -v google.golang.org/grpc 

RUN go get -u -v github.com/go-kit/kit/...
COPY . $GOPATH/src/github.com/TuneLab/go-truss/

RUN ls -la /gopath/src/github.com
RUN ls -la /gopath/src/github.com/opentracing

RUN go get -v github.com/TuneLab/go-truss/...

ENTRYPOINT ["/gopath/bin/truss"]

