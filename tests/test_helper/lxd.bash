LXC_TEST_IMAGE=jammy
DEPENDENCIES="jq"

function start_containers() {
    LXC_COUNT=$1
    LXC_BASE_NAME="microovn-test"
    ALL_CONTAINERS=""
    for (( i = 0; i < LXC_COUNT; i++ )); do
        CONTAINER_NAME="$LXC_BASE_NAME-$i"
        echo "# Launching $CONTAINER_NAME" >&3
        lxc launch -q ubuntu:$LXC_TEST_IMAGE $CONTAINER_NAME
        lxc_exec $CONTAINER_NAME "cloud-init status --wait &&
                                  export DEBIAN_FRONTEND=noninteractive &&
                                  apt update && apt install -y $DEPENDENCIES"
        ALL_CONTAINERS+="$CONTAINER_NAME "
    done

    export ALL_CONTAINERS
}

function cleanup_containers() {
    IFS=" "
    for container in $ALL_CONTAINERS ; do
        echo "# Cleaning up $container" >&3
        lxc delete --force $container
    done
}

function lxc_exec() {
    CONTAINER=$1
    CMD=$2

    lxc exec $CONTAINER -- bash -c "$CMD"

}

function lxc_file_push() {
    FILE_PATH=$1
    CONTAINER=$2

    lxc file push -q "$FILE_PATH" "$CONTAINER"
}