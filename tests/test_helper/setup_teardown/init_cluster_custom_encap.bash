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

    # Create a dedicated bridge for the east-west traffic
    network_output=$(create_lxd_network "br-east-west")
    ipv4_subnet=$(echo "$network_output" | cut -d'|' -f1)

    # Give each container an IP address from the subnet
    east_west_ips_to_containers=$(connect_containers_to_network_ipv4 "$TEST_CONTAINERS" "br-east-west" $ipv4_subnet)
    IFS=',' read -ra east_west_addrs <<< "$east_west_ips_to_containers"
    EAST_WEST_ADDRS=("${east_west_addrs[@]}")
    printf '%s\n' "${EAST_WEST_ADDRS[@]}" > "$BATS_TMPDIR/east_west_addrs.txt"

    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    local leader
    for pair in "${EAST_WEST_ADDRS[@]}"; do
        IFS='@' read -r container ip_east_west <<< "$pair"

        local addr
        addr=$(container_get_default_ip "$container" \
               "$(test_is_ipv6_test && echo inet6 || echo inet)")
        assert [ -n "$addr" ]
        if [ -z "$leader" ]; then
            microovn_init_create_cluster "$container" "$addr" "$ip_east_west" "" "" ""
            leader="$container"
        else
            local token
            token=$(microovn_cluster_get_join_token "$leader" "$container")
            microovn_init_join_cluster "$container" "$addr" "$token" "$ip_east_west" ""
        fi
    done
}

teardown_file() {
    print_diagnostics_on_failure $TEST_CONTAINERS
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
    delete_lxd_network "br-east-west"
    rm -f "$BATS_TMPDIR/east_west_addrs.txt"
}
