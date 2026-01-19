setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/tls.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 6)
    EXTERNAL_CLUSTER=""
    INTERNAL_CLUSTER=""

    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS

    local cert_path="/var/snap/microovn/common/ca.crt"
    local key_path="/var/snap/microovn/common/ca.key"
    local container_list
    read -r -a container_list <<< "$TEST_CONTAINERS"
    # Bootstrap central-only cluster on containers 1-3
    local central_leader
    local i
    for i in {0..2}; do
        local container=${container_list[$i]}
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        assert [ -n "$addr" ]
        if [ -z "$central_leader" ]; then
            # pre-generate CA shared by the external and the internal cluster
            generate_user_ca "$container" "ec" "$cert_path" "$key_path"
            microovn_init_create_cluster "$container" "$addr" "" "$cert_path" "$key_path" "central"
            central_leader="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$central_leader" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" "" "central"
        fi
        EXTERNAL_CLUSTER="$EXTERNAL_CLUSTER $container"
    done

    # Bootstrap datapath-only cluster on containers 4-6
    local leader
    for i in {3..5}; do
        local container=${container_list[$i]}
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        assert [ -n "$addr" ]
        if [ -z "$leader" ]; then
            # sync CA files
            lxc_file_transfer "$central_leader" "$cert_path" "$container" "$cert_path" 0 0
            lxc_file_transfer "$central_leader" "$key_path" "$container" "$key_path" 0 0
            microovn_init_create_cluster "$container" "$addr" "" "$cert_path" "$key_path" "switch,chassis"
            leader="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$leader" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" "" "switch,chassis"
        fi
        INTERNAL_CLUSTER="$INTERNAL_CLUSTER $container"
    done

    export TEST_CONTAINERS
    export EXTERNAL_CLUSTER
    export INTERNAL_CLUSTER
}

teardown_file() {
    print_diagnostics_on_failure $TEST_CONTAINERS
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}

