setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 3)
    export TEST_CONTAINERS
    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS

    for container in $TEST_CONTAINERS; do
        lxc_exec "$container" "microovn disable central"
        lxc_exec "$container" "microovn enable central"

        lxc_exec "$container" "microovn disable chassis"
        lxc_exec "$container" "microovn enable chassis"

        lxc_exec "$container" "microovn disable switch"
        lxc_exec "$container" "microovn enable switch"
    done

}

teardown_file() {
    print_diagnostics_on_failure $TEST_CONTAINERS
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}
