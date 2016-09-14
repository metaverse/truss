DOCKER_TRUSS_HOME := /go/src/github.com/TuneLab/go-truss
DOCKER_BUILD_IMAGE_NAME := tunelab/truss-build
DOCKER_RUN_IMAGE_NAME := tunelab/truss

MAKEFILE_PATH := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
TRUSS_HOME := $(abspath $(MAKEFILE_PATH)/..)

NOW := $(shell date -u "+%Y-%m-%dT%H_%M_%S")

# Docker target builds Truss in using the truss-build container, then
# packages the output into a new truss image, and removes the build output
# from the working directory. Note that if make targets are evaluated in
# parallel, this target will not work since it depends on the order of the
# dependent targets as specified.
docker: docker-build build-docker-run

# build-docker-build builds an image that can build Truss. It does not actually
# build truss. The resultant image must be run with the appropriate volumes
# mounted in order to build truss.
build-docker-build:
	docker build -t $(DOCKER_BUILD_IMAGE_NAME) -t $(DOCKER_BUILD_IMAGE_NAME):$(NOW) -f build/Dockerfile $(TRUSS_HOME)

# docker-build runs the build image. It builds the truss executable and pulls
# the other executables required by truss at runtime and puts everything in a
# build/ subdirectory. These executables will be used when building the runtime
# image.
docker-build: clean
	docker run --rm -v $(TRUSS_HOME):$(DOCKER_TRUSS_HOME) $(DOCKER_BUILD_IMAGE_NAME)

# build-docker-run builds the truss-run runtime image. It makes the assumption
# that the truss-build image has been run and places its output in the build/
# directory, and will place these binaries inside a runtime image.
build-docker-run:
	@if [ ! -d build ]; then echo "error: no build output directory"; exit 1; fi
	docker build -t $(DOCKER_RUN_IMAGE_NAME) -t $(DOCKER_RUN_IMAGE_NAME):$(NOW) -f run/Dockerfile $(TRUSS_HOME)

# docker-run runs truss on the .proto files specified on the command line. This
# target is mostly useful as an example, since this Makefile lives in the Truss
# source repository, and not where you'd want to actually run Truss go generate
# code.
docker-run:
	docker run --rm -v $(TRUSS_HOME):/truss $(DOCKER_RUN_IMAGE_NAME) $(filter-out $@,$(MAKECMDGOALS))

clean:
	-@rm -rf ../build 2>/dev/null


