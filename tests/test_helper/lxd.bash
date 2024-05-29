function launch_containers() {
    local containers=$*
    local image="${MICROOVN_TEST_CONTAINER_IMAGE:-ubuntu:lts}"
    for container in $containers; do
        echo "# Launching '$image' container: $container" >&3
        lxc launch -q "$image" "$container" < \
            "$BATS_TEST_DIRNAME/lxd-instance-config.yaml"
    done
}

function wait_containers_ready() {
    local containers=$*

    for container in $containers; do
        echo "# Waiting for $container to be ready" >&3
        lxc_exec "$container" "cloud-init status --wait &&
                               snap wait system seed.loaded"
    done
}

function delete_containers() {
    local containers=$*

    for container in $containers; do
        echo "# Cleaning up $container" >&3
        lxc delete --force "$container"
    done
}

function lxc_exec() {
    local container=$1; shift
    local cmd=$1; shift

    lxc exec "$container" -- bash -c "$cmd"

}

function lxc_file_push() {
    local file_path=$1; shift
    local container_path=$1; shift

    lxc file push -q "$file_path" "$container_path"
}
