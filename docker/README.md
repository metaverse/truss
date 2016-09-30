# Truss and Docker

Truss can run inside Docker. Truss' Docker strategy is two parts: a docker
image to build truss, and a docker image that runs truss.

## Why Two Docker Images?

Most people will never need to build Truss, and the dependencies needed to
build truss are numerous and somewhat large. The truss-build image weighs in at
over a gigabyte because of this. Running truss requires less, though: just the
truss binary, the protoc binary and protoc plugin binaries.

## I don't want to build Docker images.

You don't have to. We have a Docker image you can use. Just pull
`tunelab/truss`, and run the Truss that's been built there.

## How Can I Run Truss in Docker?

Given a truss runtime image, run a container with your work directory mounted
and set as the working directory, like so:
```
$ docker run --rm -v $PWD:/truss tunelab/truss myservice.proto
```
This will run Truss with the current directory mounted into the container as
`/truss`, which is this image is configured to use as its working directory.
Note that if you use a container mount point other than `/truss`, then you will
need to specify the working directory when running the container, since it has
diverged from the default. For example:

```
$ docker run --rm -v $PWD:/build -w /build tunelab/truss myservice.proto
```
Truss won't work unless its working directory is where the `.proto` files are
mounted. So either be sure to mount to `/truss`, or be sure to specify the
working directory with the `-w` switch.

## I Want to Build a Truss Docker Image

No problem. The `Makefile` in the project root has targets that will do this
for you. First you'll need to build a truss-build image:
```
$ make build-docker-build
```
This will take several minutes as it pulls various dependencies from the internet.
When this finishes, you'll have a Docker image capable of building Truss. Then,
from the project root, run the build image that was just created:
```
$ make docker-build
```
This will mount the project into the build container, build truss, and place it and
other dependencies that Truss needs at runtime into a `build` directory beneath the
project root. The contents of this directory will be used to create the actual Truss
runtime image:
```
$ make build-docker-run
```
This will yield a new `tunelab/truss` image. You can use a different tag for the image
by specifying `DOCKER_RUN_IMAGE_NAME` during the make invocation. For example,
```
$ make DOCKER_RUN_IMAGE_NAME=bobbys-truss build-docker-run
```
will generate Truss with in image tagged `bobbys-truss`. This is useful if you
want to push it to Docker Hub, since you won't be able to push to
`tunelab/truss`.

