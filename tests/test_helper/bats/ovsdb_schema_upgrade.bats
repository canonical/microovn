# This is a bash shell fragment -*- bash -*-

# Define test filename prefix that helps to determine from which version should
# the upgrade be tested.
export TEST_NAME_PREFIX="ovsdb_schema_upgrade"

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/lxd.bash
    load test_helper/common.bash
    load test_helper/microovn.bash
    load test_helper/ovsdb.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

@test "Test OVSDB cluster schema upgrade" {
    echo "# Checking if SB or NB database schema changed from MicroOVN rev. $MICROOVN_SNAP_REV" >&3
    local original_nb=""
    local original_sb=""
    local target_nb=""
    local target_sb=""
    local probe_container=""

    read -r -a containers_to_upgrade <<< "$TEST_CONTAINERS"
    probe_container="${containers_to_upgrade[0]}"

    # Get currently running OVN schema versions
    original_nb=$(get_current_ovsdb_schema_version "$probe_container" "nb")
    original_sb=$(get_current_ovsdb_schema_version "$probe_container" "sb")
    # Get schema versions that are included in the tested MicroOVN snap
    target_nb=$(get_ovsdb_schema_version_from_snap "$probe_container" "$MICROOVN_SNAP_PATH" "nb")
    target_sb=$(get_ovsdb_schema_version_from_snap "$probe_container" "$MICROOVN_SNAP_PATH" "sb")


    # Skip this test if there's no change between schemas in running MicroOVN version
    # (installed from snap store) and tested MicroOVN version (installed from source)
    if [ "$original_nb" == "$target_nb" ] && [ "$original_sb" == "$target_sb" ]; then
        skip "OVN database schemas did not change. Skipping this check"
    fi

    # Based on  the version of original MicroOVN snap, it may not have necessary API to
    # report expected schema version. In that case we'll expect an error message from these members.
    # Otherwise we expect them to report old schema version
    local old_member_msg_nb=""
    local old_member_msg_sb=""
    if [ "$MICROOVN_SNAP_REV" -le 395 ]; then
        old_member_msg_nb="Missing API. MicroOVN needs upgrade"
        old_member_msg_sb="Missing API. MicroOVN needs upgrade"
    else
        old_member_msg_nb="$original_nb"
        old_member_msg_sb="$original_sb"
    fi

    local container_index=0
    local upgrade_container_index=0
    local old_container_index=0
    local containers_size="${#containers_to_upgrade[@]}"
    local upgraded_size=0

    # Loop below detects and performs internal dqlite schema upgrade if necessary.
    # When upgrading dqlite, all nodes have to be upgraded before cluster becomes
    # usable again.
    for ((container_index = 0; container_index < "$containers_size"; container_index++)) do
        local container="${containers_to_upgrade[$container_index]}"
        # Upgrade MicroOVN cluster, one cluster member at a time.
        install_microovn "$MICROOVN_SNAP_PATH" "$container"
        upgraded_size=$(( "$upgraded_size" + 1 ))

        # After the first host is upgraded, we can try to determine whether internal Dqlite schema upgrade
        # is required.
        if [ "$container_index" -eq 0 ]; then
            wait_ovn_services "$container"
            run wait_microovn_online "$container" 30
            # If all nodes show up as ONLINE we can assume that Dqlite upgrade is not necessary
            if [ "$status" = 0 ] ; then
                echo "# ($container) reports all systems are reachable. No dqlite schema update detected" >&3
                break
            fi
        fi

        # If we upgraded all systems because of a dqlite schema update, ensure all nodes are online.
        if [ "$upgraded_size" -eq "$containers_size" ] ; then
            run wait_microovn_online "$container" 65
            assert_success
        else
            # check that the schema upgrade was not triggered yet if we have not updated all nodes.
            local current_sb=""
            local current_nb=""
            current_nb=$(get_current_ovsdb_schema_version "$probe_container" "nb")
            current_sb=$(get_current_ovsdb_schema_version "$probe_container" "sb")
            assert [ "$current_nb" == "$original_nb" ]
            assert [ "$current_sb" == "$original_sb" ]
        fi
    done

    # If the loop above did not upgrade all the nodes in the deployment, loop below walks through the gradual upgrade
    # of all the nodes, performing checks to make sure that OVSDB schema upgrade completed succesfully.
    for ((container_index = 0; container_index < "$containers_size"; container_index++)) do
        # If the current container index is larger or equal to the upgraded size, then we haven't run the update for that node.
        if [ "$container_index" -ge  "$upgraded_size" ]; then
            local container="${containers_to_upgrade[$container_index]}"
            # Upgrade MicroOVN cluster, one cluster member at a time.
            echo "# Upgrading ($container) before ovsdb schema verification"
            install_microovn "$MICROOVN_SNAP_PATH" "$container"
            upgraded_size=$(( "$upgraded_size" + 1 ))

            wait_ovn_services "$container"
            wait_microovn_online "$container" 30
        fi

        # break out of the loop if every container has been upgraded
        if [ $(( "$container_index" + 1 )) -eq  "$containers_size" ] || [ "$upgraded_size" -eq "$containers_size" ]; then
            break
        fi

        local status=""
        status=$(lxc_exec "$container" "microovn status")

        # Verify that upgraded members report the new expected schema versions
        echo -e "Current status:\n $status"
        for ((upgrade_container_index = 0; upgrade_container_index <= container_index; upgrade_container_index++)); do
            local upgraded_container="${containers_to_upgrade[$upgrade_container_index]}"

            echo "Checking status for '$upgraded_container: $target_nb'"
            grep "$upgraded_container: $target_nb" <<< "$status" > /dev/null
            echo "# $upgraded_container now expects NB schema version: $target_nb" >&3

            echo "Checking status for '$upgraded_container: $target_sb'"
            grep "$upgraded_container: $target_sb" <<< "$status" > /dev/null
            echo "# $upgraded_container now expects SB schema version: $target_sb" >&3
        done

        # Verify that members, that are not upgraded yet, report either old version, or an error
        # in case that they don't support required API at all.
        for ((old_container_index = container_index + 1; old_container_index < "$containers_size"; old_container_index++)) do
            local old_container="${containers_to_upgrade[$old_container_index]}"

            echo "Checking status for '$old_container: $old_member_msg_sb'"
            grep "$old_container: $old_member_msg_sb" <<< "$status" > /dev/null

            echo "Checking status for '$old_container: $old_member_msg_nb'"
            grep "$old_container: $old_member_msg_nb" <<< "$status" > /dev/null

            echo "# $old_container is still awaiting upgrade" >&3
        done

        # check that the schema upgrade was not triggered yet.
        local current_sb=""
        local current_nb=""
        current_nb=$(get_current_ovsdb_schema_version "$probe_container" "nb")
        current_sb=$(get_current_ovsdb_schema_version "$probe_container" "sb")
        assert [ "$current_nb" == "$original_nb" ]
        assert [ "$current_sb" == "$original_sb" ]
    done

    # After all cluster members are upgraded, verify that the cluster reports upgraded schema version
    # as well.
    local timeout=30
    for (( i = 0; i < "$timeout"; i++ )); do
        echo "# Waiting for Southbound and Northbound databases to finish schema upgrade ($i/$timeout)"
        local err=0

        run lxc_exec "$probe_container" "microovn status | grep 'OVN Southbound: OK ($target_sb)' "
        if [ "$status" -ne 0 ]; then
            err=1
            echo "# OVN Southbound database schema is not upgraded yet"
        fi

        run lxc_exec "$probe_container" "microovn status | grep 'OVN Northbound: OK ($target_nb)' "
        if [ "$status" -ne 0 ]; then
            err=1
            echo "# OVN Northbound database schema is not upgraded yet"
        fi

        if [ "$err" -eq 0 ]; then
            break
        fi

        sleep 1
    done

    if [ "$err" -ne 0 ]; then
        echo "# OVN schema upgraded did not finish in expected timeout"
        return 1
    fi

    # check that the schema upgrade was triggered.
    local current_sb=""
    local current_nb=""
    current_nb=$(get_current_ovsdb_schema_version "$probe_container" "nb")
    current_sb=$(get_current_ovsdb_schema_version "$probe_container" "sb")
    assert [ "$current_nb" == "$target_nb" ]
    assert [ "$current_sb" == "$target_sb" ]

    echo "# OVN database schema upgrade finished" >&3
}
