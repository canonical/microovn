function install_microovn() {
    local snap_file=$1; shift
    local containers=$*

    for container in $containers; do
        echo "# Deploying MicroOVN to $container" >&3
        lxc_file_push "$snap_file" "$container/tmp/microovn.snap"
        echo "# Installing MicroOVN in container $container" >&3
        lxc_exec "$container" "snap install /tmp/microovn.snap --dangerous"
        echo "# Connecting plugs in container $container" >&3
        lxc_exec "$container" "for plug in firewall-control \
                                           hardware-observe \
                                           hugepages-control \
                                           network-control \
                                           openvswitch-support \
                                           process-control \
                                           system-trace; do \
                                   sudo snap connect microovn:\$plug;done"
    done
}

function microovn_cluster_get_join_token() {
    local existing_member=$1; shift
    local new_member=$1; shift

    lxc_exec "$existing_member" "microovn cluster add $new_member"
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

function microovn_init_create_cluster() {
    local container=$1; shift
    local address=$1; shift

    cat << EOF | lxc_exec "$container" "expect -"
spawn "sudo" "microovn" "init"

expect "Please choose the address MicroOVN will be listening on" {
    send "$address\n"
}

expect "Would you like to create a new MicroOVN cluster?" {
    send "yes\n"
}

expect "Please choose a name for this system" {
    send "\n"
}

expect "Would you like to add additional servers to the cluster?" {
    send "no\n"
}

expect eof
EOF
}

function microovn_init_join_cluster() {
    local container=$1; shift
    local address=$1; shift
    local token=$1; shift
    cat << EOF | lxc_exec "$container" "expect -"
spawn "sudo" "microovn" "init"

expect "Please choose the address MicroOVN will be listening on" {
    send "$address\n"
}

expect "Would you like to create a new MicroOVN cluster?" {
    send "no\n"
}

expect "Please enter your join token:" {
    send "$token\n"
}

expect eof
EOF
}

function microovn_get_cluster_address() {
    local container=$1; shift

    lxc_exec "$container" "microovn status" | \
        awk -F\( "/$container/{sub(/\)\$/,\"\");print\$2}"
}
