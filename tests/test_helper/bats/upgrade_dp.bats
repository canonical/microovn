# This is a bash shell fragment -*- bash -*-

# The test simulates instance(s) by creating Linux namespaces inside the test
# containers.
export TEST_LXD_LAUNCH_ARGS="-c security.nesting=true"

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/lxd.bash
    load test_helper/common.bash
    load test_helper/microovn.bash
    load test_helper/upgrade_procedures.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]

    # Set up namespace for data path test
    for container in $CHASSIS_CONTAINERS; do
        microovn_add_gw_router "$container"
        netns_add "$container" upgrade_dp0
        microovn_add_vif "$container" upgrade_dp0 upgrade_vif0
    done
}

teardown() {
    # Remove namespace used for data path test
    for container in $CHASSIS_CONTAINERS; do
        microovn_delete_vif "$container" upgrade_dp0 upgrade_vif0
        netns_delete "$container" upgrade_dp0
        microovn_delete_gw_router "$container"
    done
}


@test "MicroOVN control of schema conversion prevents data path downtime" {
    for container in $CHASSIS_CONTAINERS; do
        ping_start "$container" 10.42.4.1 upgrade_dp0
    done

    echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV on " >&3
    install_microovn "$MICROOVN_SNAP_PATH" $CENTRAL_CONTAINERS


    for container in $CENTRAL_CONTAINERS; do
        function sbstatus() {
            lxc_exec "$container" 'microovn status | grep -q OVN\ Southbound:\ Upgrade'
        }
        function nbstatus() {
            lxc_exec "$container" 'microovn status | grep -q OVN\ Northbound:\ Upgrade'
        }
        wait_until sbstatus
        wait_until nbstatus
        break
    done

    for container in $CHASSIS_CONTAINERS; do
        n_lost=$(ping_reap "$container" 10.42.4.1 upgrade_dp0 | \
                     awk '/packets/{print$1-$4}')

        # Apart from the one packet that can be lost while stopping ``ping``,
        # we expect zero loss.
        assert test "$n_lost" -le 1
    done

    for container in $CHASSIS_CONTAINERS; do
        ping_start "$container" 10.42.4.1 upgrade_dp0
    done

    echo "# Upgrading MicroOVN from revision $MICROOVN_SNAP_REV on " >&3
    install_microovn "$MICROOVN_SNAP_PATH" $CHASSIS_CONTAINERS

    for container in $CENTRAL_CONTAINERS; do
        function sbstatus() {
            lxc_exec "$container" 'microovn status | grep -q OVN\ Southbound:\ OK'
        }
        function nbstatus() {
            lxc_exec "$container" 'microovn status | grep -q OVN\ Northbound:\ OK'
        }
        wait_until sbstatus
        wait_until nbstatus
        break
    done

    for container in $CHASSIS_CONTAINERS; do
        n_lost=$(ping_reap "$container" 10.42.4.1 upgrade_dp0 | \
                     awk '/packets/{print$1-$4}')

        run lxc_exec "$container" journalctl

        # Upgrading the node we are monitoring will inevitably cause some data
        # path interruption.
        assert test "$n_lost" -eq 0
    done
}
