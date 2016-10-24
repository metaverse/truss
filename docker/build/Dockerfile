# Truss
# truss-build Docker image
#
# NOTE: YOU WILL PROBABLY NEVER NEED TO CREATE A TRUSS-BUILD IMAGE. The primary use
# of the truss-build image will be to create updated truss-run images. A pre-built
# truss-build image from the Docker registry can build the binaries that are
# required for a Truss runtime image. Or more simply, a pre-built truss-run image
# can be used, if it meets your needs.
#
# This Dockerfile builds an image that can itself build Truss. The container
# that runs this image must mount the project source code to an external
# volume, so the go compiler can build it should have a volume mounted, so it
# can read the Truss code and generate binaries.
#
# The base of this Docker image is a full Linux distribution, because the protoc
# binary is dynamically linked and requires libraries that are not present in a
# minimal distribution like Alpine. We pull the required libraries from this base
# image and package them up to be added to the runtime image.
#
# E.g., docker run --rm -v $PWD:/go/src/github.com/TuneLab/go-truss tunelab/truss-build
#
# The binaries are subsequently packaged into a Docker image for running Truss.
# The runtime images is much smaller because of this.
FROM golang:1.7.1
MAINTAINER lab@tune.com

ENV GOSU_VERSION 1.9
ENV GOSU_DOWNLOAD_URL https://github.com/tianon/gosu/releases/download/$GOSU_VERSION
ENV PROTOBUF_VERSION 3.0.2
ENV PROTOBUF_ZIP protoc-$PROTOBUF_VERSION-linux-x86_64.zip
ENV TRUSS_REPO github.com/TuneLab/go-truss
ENV TRUSS_PATH $GOPATH/src/$TRUSS_REPO
ENV PROTO_GEN_TRUSS protoc-gen-truss-protocast

WORKDIR $TRUSS_PATH

# Get all the supporting libraries and programs necessary to build truss.
# Get the Protobuf compiler and unzip it into the build output directory.
# Get the gosu tool that will allow us to render files into the mounted
# directory as the user who invoked the command, rather than root.
# Finally, clean up after ourselves.
RUN set -x \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget unzip git \
    && update-ca-certificates \
    && go version \
    && go get -u -v github.com/lightstep/lightstep-tracer-go \
        github.com/opentracing/opentracing-go \
        github.com/openzipkin/zipkin-go-opentracing \
        github.com/prometheus/client_golang/prometheus \
        github.com/sourcegraph/appdash/opentracing \
        github.com/golang/protobuf/proto \
        github.com/golang/protobuf/protoc-gen-go \
        github.com/go-kit/kit/... \
    \
    && wget -O /tmp/$PROTOBUF_ZIP https://github.com/google/protobuf/releases/download/v$PROTOBUF_VERSION/$PROTOBUF_ZIP \
    && (cd / && unzip /tmp/$PROTOBUF_ZIP bin/protoc && chmod 755 /bin/protoc) \
    && ldd /bin/protoc | egrep -o '/.+ ' | sed -e 's/^\///' > /tmp/protoc-libs.txt \
    \
    && DPKG_ARCH=$(dpkg --print-architecture | awk -F- '{ print $NF }') \
    && wget -O /usr/local/bin/gosu $GOSU_DOWNLOAD_URL/gosu-$DPKG_ARCH \
    && wget -O /usr/local/bin/gosu.asc $GOSU_DOWNLOAD_URL/gosu-$DPKG_ARCH.asc \
    && export GNUPGHOME="$(mktemp -d)" \
    && gpg --keyserver ha.pool.sks-keyservers.net --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4 \
    && gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu \
    && chmod +x /usr/local/bin/gosu \
    && gosu nobody true \
    \
    && rm -r "$GNUPGHOME" /usr/local/bin/gosu.asc /tmp/$PROTOBUF_ZIP \
    && apt-get purge -y --auto-remove ca-certificates wget unzip git

COPY docker/build/*.sh /

ENTRYPOINT ["bash"]
CMD ["-c", "/build-wrapper.sh $TRUSS_PATH /build-truss.sh"]

# After execution of this image in container, $GOPATH/build contents will be on
# locally mounted volume, and can be placed into the Truss runtime image.
