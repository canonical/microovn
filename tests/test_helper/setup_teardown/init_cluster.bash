setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 3)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    local leader
    for container in $TEST_CONTAINERS; do
        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        assert [ -n "$addr" ]
        if [ -z "$leader" ]; then
            microovn_init_create_cluster "$container" "$addr" ""
            leader="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$leader" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" ""
        fi
    done
}

teardown_file() {
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}

