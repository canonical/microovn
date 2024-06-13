setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/upgrade_procedures.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    ABS_TOP_TEST_DIRNAME="${BATS_TEST_DIRNAME}/"
    export ABS_TOP_TEST_DIRNAME

    # Env variable MICROOVN_SNAP_CHANNEL must be specified for tests to know
    # from which channel should be MicroOVN installed before upgrading
    assert [ -n "$MICROOVN_SNAP_CHANNEL" ]

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

    # detect and export initial MicroOVN snap revision
    container=$(echo "$TEST_CONTAINERS" | awk '{print $1;}' )
    export MICROOVN_SNAP_REV=""
    MICROOVN_SNAP_REV=$(lxc_exec "$container" "snap list | grep microovn | awk '{print \$3;}'")
    assert [ -n "$MICROOVN_SNAP_REV" ]

    if [ -n "$UPGRADE_DO_UPGRADE" ]; then
        echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV" >&3
        install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS

        for container in $TEST_CONTAINERS; do
            local container_services
            container_services=$(microovn_get_cluster_services "$container")
            if [[ "$container_services" != *"central"* ]]; then
                continue
            fi
            microovn_wait_ovndb_state "$container" nb connected 15
            microovn_wait_ovndb_state "$container" sb connected 15
        done

        perform_manual_upgrade_steps $TEST_CONTAINERS
    fi
}

teardown_file() {
    delete_containers $TEST_CONTAINERS
}
