function _wait_for_snapd() {
    local container=$1; shift

    echo "# Waiting for snapd to be ready on $container" >&3
    lxc_exec "$container" "snap wait system seed.loaded"
}

function install_microovn() {
    local snap_file=$1; shift
    local containers=$*

    for container in $containers; do
        echo "# Deploying MicroOVN to $container" >&3
        lxc_file_push "$snap_file" "$container/tmp/microovn.snap"
        _wait_for_snapd "$container"
        echo "# Installing MicroOVN in container $container" >&3
        lxc_exec "$container" "snap install /tmp/microovn.snap --dangerous"
    done
}

function bootstrap_cluster() {
    local leader=""
    local containers=$*

    for container in $containers; do
        if [ -z "$leader" ]; then
            echo "# Bootstrapping MicroOVN on $container" >&3
            lxc_exec "$container" "microovn cluster bootstrap"
            leader="$container"
            continue
        fi

        echo "# Adding $container to the cluster" >&3
        local token
        token=$(lxc_exec "$leader" "microovn cluster add $container")
        echo "# Joining cluster with $container" >&3
        lxc_exec "$container" "microovn cluster join $token"
    done
}
