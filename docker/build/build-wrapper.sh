#!/bin/sh
#
# This script is a wrapper to execute the build-truss.sh script potentially as
# another user. This is useful for preventing root ownership of files created
# inside the container, but written to a volume mounted from the host. The
# files will instead be owned by the user that Docker reports as owning the
# bind-mounted directory.
#
# This approach was guided by the following resources:
# * https://denibertovic.com/posts/handling-permissions-with-docker-volumes/
# * https://stackoverflow.com/questions/30052019/docker-creates-files-as-root-in-mounted-volume
# * http://stackoverflow.com/a/27925525/602137
set -ex

WORK_DIR="$1"
BUILD_UID=$(stat -c %u $WORK_DIR)
BUILD_GID=$(stat -c %g $WORK_DIR)
groupadd -fg $BUILD_GID buildgrp
useradd --create-home --uid $BUILD_UID --gid $BUILD_GID buildusr

fix_owner () {
    chown -R $BUILD_UID:$BUILD_GID "$WORK_DIR"
}

trap fix_owner EXIT

shift
cd $WORK_DIR

/usr/local/bin/gosu $BUILD_UID:$BUILD_GID "$@"

