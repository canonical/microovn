function launch_containers() {
    local image_name=$1; shift
    local deps=$1; shift
    local containers=$*

    for container in $containers; do
        echo "# Launching $container" >&3
        lxc launch -q "ubuntu:$image_name" "$container"
        lxc_exec "$container" "cloud-init status --wait &&
                             export DEBIAN_FRONTEND=noninteractive &&
                             apt update && apt install -y $deps"
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
