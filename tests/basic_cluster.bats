setup_file() {
    load test_helper/lxd.bash
    load test_helper/microovn.bash

    start_containers 3
    install_microovn $MICROOVN_SNAP_PATH $ALL_CONTAINERS
    bootstrap_cluster $ALL_CONTAINERS
}

teardown_file() {
    cleanup_containers
}

setup() {
    load test_helper/lxd.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash
}

@test "Expected MicroOVN cluster count" {
    # Extremely simplified check that cluster has required number of members
    FIPS=" "
    for container in $ALL_CONTAINERS ; do
        echo "Checking cluster members on $container"
        run lxc_exec $container "microovn cluster list --format json | jq length"
        assert_output "3"
    done
}

@test "Expected services up" {
    # Check that all expected services are active on cluster members
    SERVICES="snap.microovn.central snap.microovn.chassis snap.microovn.daemon snap.microovn.switch"
    FIPS=" "
    for container in $ALL_CONTAINERS ; do
        for service in $SERVICES ; do
            echo "Checking status of $service on $container"
            run lxc_exec $container "systemctl is-active $service"
            assert_output "active"
        done
    done
}
