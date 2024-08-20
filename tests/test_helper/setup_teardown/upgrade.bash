setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/upgrade_procedures.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    ABS_TOP_TEST_DIRNAME="${BATS_TEST_DIRNAME}/"
    export ABS_TOP_TEST_DIRNAME

    # Determine MicroOVN channel from which we are upgrading
    test_name=$(basename "$BATS_TEST_FILENAME")
    MICROOVN_SNAP_CHANNEL=$(get_upgrade_test_version "$test_name" "$TEST_NAME_PREFIX")

    # Create test deployment
    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 4)
    CENTRAL_CONTAINERS=""
    CHASSIS_CONTAINERS=""

    export TEST_CONTAINERS
    export CENTRAL_CONTAINERS
    export CHASSIS_CONTAINERS

    launch_containers $TEST_CONTAINERS
    wait_containers_ready $TEST_CONTAINERS
    install_microovn_from_store "$MICROOVN_SNAP_CHANNEL" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS

    # Categorize containers as "CENTRAL" and "CHASSIS" based on the services they run
    local container=""
    for container in $TEST_CONTAINERS; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" == *"central"* ]]; then
            CENTRAL_CONTAINERS+="$container "
        else
            CHASSIS_CONTAINERS+="$container "
        fi
    done

    # Make sure that microcluster is fully converged before proceeding.
    # Performing further actions before the microcluster is ready may lead to
    # unexpectedly long convergence after a microcluster schema upgrade.
    for container in $TEST_CONTAINERS; do
        wait_microovn_online "$container" 60
    done

    # detect and export initial MicroOVN snap revision
    container=$(echo "$TEST_CONTAINERS" | awk '{print $1;}' )
    export MICROOVN_SNAP_REV=""
    MICROOVN_SNAP_REV=$(lxc_exec "$container" "snap list | grep microovn | awk '{print \$3;}'")
    assert [ -n "$MICROOVN_SNAP_REV" ]

    if [ -n "$UPGRADE_DO_UPGRADE" ]; then
        assert [ -n "$CENTRAL_CONTAINERS" ]
        assert [ -n "$CHASSIS_CONTAINERS" ]
        echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV " \
             "central container(s)." >&3
        install_microovn "$MICROOVN_SNAP_PATH" $CENTRAL_CONTAINERS

        for container in $CENTRAL_CONTAINERS; do
            microovn_wait_ovndb_state "$container" nb connected 15
            microovn_wait_ovndb_state "$container" sb connected 15
        done

        maybe_perform_manual_upgrade_steps $CENTRAL_CONTAINERS

        echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV " \
             "on chassis container(s)." >&3
        install_microovn "$MICROOVN_SNAP_PATH" $CHASSIS_CONTAINERS

        wait_microovn_online "$container" 60
    fi
}

teardown_file() {
    collect_coverage $TEST_CONTAINERS
    delete_containers $TEST_CONTAINERS
}
