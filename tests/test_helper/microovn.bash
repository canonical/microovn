
function _wait_for_snapd() {
    CONTAINER=$1

    echo "# Waiting for snapd to be ready on $CONTAINER" >&3
    lxc_exec $CONTAINER "snap wait system seed.loaded"
}

function install_microovn() {
    SNAP_FILE=$1
    shift

    IFS=" "
    for container in "$@" ; do
        echo "# Deploying MicroOVN to $container" >&3
        lxc_file_push "$SNAP_FILE" "$container/tmp/microovn.snap"
        _wait_for_snapd $container
        echo "# Installing MicroOVN in container $container" >&3
        lxc_exec $container "snap install /tmp/microovn.snap --dangerous"
    done
}

function bootstrap_cluster() {
    LEADER=""

    for container in "$@" ; do
        if [ -z "$LEADER" ]; then
            echo "# Bootstrapping MicroOVN on $container" >&3
            lxc_exec $container "microovn cluster bootstrap"
            LEADER="$container"
            continue
        fi

        echo "# Adding $container to the cluster" >&3
        TOKEN=$(lxc_exec $LEADER "microovn cluster add $container")
        echo "# Joining cluster with $container" >&3
        lxc_exec $container "microovn cluster join $TOKEN"
    done
}
