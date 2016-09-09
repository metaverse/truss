# Base Docker Image
FROM alpine:3.4

RUN \
apk add --no-cache git go openssh && \
apk add --update --no-cache unzip build-base autoconf automake libtool libstdc++ curl

RUN git clone https://github.com/google/protobuf --depth 1

RUN	\
cd /protobuf && \
./autogen.sh && \
./configure --prefix=/usr && \
make && \
make install

RUN \
rm -rf protobuf && \
cd /

ENV GOROOT /usr/lib/go
ENV GOPATH /gopath
ENV GOBIN /gopath/bin
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

RUN \
go get -u -v github.com/lightstep/lightstep-tracer-go && \
go get -u -v github.com/opentracing/opentracing-go && \
go get -u -v github.com/openzipkin/zipkin-go-opentracing && \
go get -u -v github.com/prometheus/client_golang/prometheus && \
go get -u -v github.com/sourcegraph/appdash/opentracing && \
go get -u -v golang.org/x/net/context && \
go get -u -v sourcegraph.com/sourcegraph/appdash && \
go get -u -v github.com/golang/protobuf/proto && \
go get -u -v github.com/golang/protobuf/protoc-gen-go && \
go get -u -v google.golang.org/grpc && \
go get -u -v github.com/go-kit/kit/...

COPY . $GOPATH/src/github.com/TuneLab/go-truss/

RUN go get -v github.com/TuneLab/go-truss/...

ENTRYPOINT ["/gopath/bin/truss"]
