setup_file() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash


    TEST_CONTAINERS=$(container_names "$BATS_TEST_FILENAME" 4)
    CENTRAL_CONTAINERS=""
    CHASSIS_CONTAINERS=""

    export TEST_CONTAINERS
    export CENTRAL_CONTAINERS
    export CHASSIS_CONTAINERS
    launch_containers jammy $TEST_CONTAINERS
    install_microovn "$MICROOVN_SNAP_PATH" $TEST_CONTAINERS
    bootstrap_cluster $TEST_CONTAINERS

    # Categorize containers as "CENTRAL" and "CHASSIS" based on the services they run
    for container in $TEST_CONTAINERS; do
        container_services=$(microovn_get_cluster_services "$container")
        if [[ "$container_services" == *"central"* ]]; then
            CENTRAL_CONTAINERS+="$container "
        else
            CHASSIS_CONTAINERS+="$container "
        fi
    done

}

teardown_file() {
    delete_containers $TEST_CONTAINERS
}

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/tls.bash
    load test_helper/microovn.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
    assert [ -n "$CENTRAL_CONTAINERS" ]
    assert [ -n "$CHASSIS_CONTAINERS" ]
}

