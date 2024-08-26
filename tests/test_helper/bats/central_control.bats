load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash


    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

central_register_test_functions() {
    bats_test_function \
        --description "Testing of central node migration" \
        -- central_tests
}

# this takes a space seperated array of containers and then disables central
# services on the first two nodes, leaving a cluster of one, enables central on
# the 4th node then disabling central on the 3rd node. leaving again a cluster
# of one but without any original members, then two more members are added to
# the central cluster both of which were not original members.
#
# Then this function tests the correct services being enabled to ensure its
# worked
central_tests() {
    read -r -a containers_to_upgrade <<< "$TEST_CONTAINERS"

    local ctl_path="/var/snap/microovn/common/run/ovn/"

    target_server_id_nb=$(microovn_ovndb_server_id ${containers_to_upgrade[0]} "nb")
    target_server_id_sb=$(microovn_ovndb_server_id ${containers_to_upgrade[0]} "sb")
    run lxc_exec "${containers_to_upgrade[0]}" "microovn disable central"
    assert_output -p "Service central disabled"

    # check that container has left
    wait_ovsdb_cluster_container_leave "$target_server_id_nb" "$ctl_path/ovnnb_db.ctl" "OVN_Northbound" 30 "${containers_to_upgrade[1]} ${containers_to_upgrade[2]}"
    wait_ovsdb_cluster_container_leave "$target_server_id_sb" "$ctl_path/ovnsb_db.ctl" "OVN_Southbound" 30 "${containers_to_upgrade[1]} ${containers_to_upgrade[2]}"

    target_server_id_nb=$(microovn_ovndb_server_id ${containers_to_upgrade[1]} "nb")
    target_server_id_sb=$(microovn_ovndb_server_id ${containers_to_upgrade[1]} "sb")
    run lxc_exec "${containers_to_upgrade[1]}" "microovn disable central"
    assert_output -p "Service central disabled"

    # check that container has left
    wait_ovsdb_cluster_container_leave "$target_server_id_nb" "$ctl_path/ovnnb_db.ctl" "OVN_Northbound" 30 "${containers_to_upgrade[2]}"
    wait_ovsdb_cluster_container_leave "$target_server_id_sb" "$ctl_path/ovnsb_db.ctl" "OVN_Southbound" 30 "${containers_to_upgrade[2]}"

    run lxc_exec "${containers_to_upgrade[3]}" "microovn enable central"
    assert_output -p "Service central enabled"

    # check that container has joined
    wait_ovsdb_cluster_container_join "$(microovn_ovndb_server_id "${containers_to_upgrade[3]}" "nb")" "$ctl_path/ovnnb_db.ctl" "OVN_Northbound" 30 "${containers_to_upgrade[2]}"
    wait_ovsdb_cluster_container_join "$(microovn_ovndb_server_id "${containers_to_upgrade[3]}" "sb")" "$ctl_path/ovnsb_db.ctl" "OVN_Southbound" 30 "${containers_to_upgrade[2]}"

    target_server_id_nb=$(microovn_ovndb_server_id ${containers_to_upgrade[2]} "nb")
    target_server_id_sb=$(microovn_ovndb_server_id ${containers_to_upgrade[2]} "sb")
    run lxc_exec "${containers_to_upgrade[2]}" "microovn disable central"
    assert_output -p "Service central disabled"

    # check that container has left
    wait_ovsdb_cluster_container_leave "$target_server_id_nb" "$ctl_path/ovnnb_db.ctl" "OVN_Northbound" 30 "${containers_to_upgrade[3]}"
    wait_ovsdb_cluster_container_leave "$target_server_id_sb" "$ctl_path/ovnsb_db.ctl" "OVN_Southbound" 30 "${containers_to_upgrade[3]}"

    run lxc_exec "${containers_to_upgrade[4]}" "microovn enable central"
    assert_output -p "Service central enabled"

    # check that container has joined
    wait_ovsdb_cluster_container_join "$(microovn_ovndb_server_id "${containers_to_upgrade[4]}" "nb")" "$ctl_path/ovnnb_db.ctl" "OVN_Northbound" 30 "${containers_to_upgrade[3]}"
    wait_ovsdb_cluster_container_join "$(microovn_ovndb_server_id "${containers_to_upgrade[4]}" "sb")" "$ctl_path/ovnsb_db.ctl" "OVN_Southbound" 30 "${containers_to_upgrade[3]}"

    run lxc_exec "${containers_to_upgrade[5]}" "microovn enable central"
    assert_output -p "Service central enabled"

    # check that container has joined
    wait_ovsdb_cluster_container_join "$(microovn_ovndb_server_id "${containers_to_upgrade[5]}" "nb")" "$ctl_path/ovnnb_db.ctl" "OVN_Northbound" 30 ${containers_to_upgrade[3]} ${containers_to_upgrade[4]}
    wait_ovsdb_cluster_container_join "$(microovn_ovndb_server_id "${containers_to_upgrade[5]}" "sb")" "$ctl_path/ovnsb_db.ctl" "OVN_Southbound" 30 ${containers_to_upgrade[3]} ${containers_to_upgrade[4]}

    assert [ -z "$(run lxc_exec "${containers_to_upgrade[0]}" "microovn status | grep -ozE 'microovn-central-control-1[^-]*' | grep central")"]
    assert [ -z "$(run lxc_exec "${containers_to_upgrade[0]}" "snap services microovn | grep ovn-northd | grep enabled")"]
    assert [ -z "$(run lxc_exec "${containers_to_upgrade[1]}" "microovn status | grep -ozE 'microovn-central-control-2[^-]*' | grep central")"]
    assert [ -z "$(run lxc_exec "${containers_to_upgrade[1]}" "snap services microovn | grep ovn-northd | grep enabled")"]
    assert [ -z "$(run lxc_exec "${containers_to_upgrade[2]}" "microovn status | grep -ozE 'microovn-central-control-3[^-]*' | grep central")"]
    assert [ -z "$(run lxc_exec "${containers_to_upgrade[2]}" "snap services microovn | grep ovn-northd | grep enabled")"]
    assert [ -n "$(run lxc_exec "${containers_to_upgrade[3]}" "microovn status | grep -ozE 'microovn-central-control-4[^-]*' | grep central")"]
    assert [ -n "$(run lxc_exec "${containers_to_upgrade[3]}" "snap services microovn | grep ovn-northd | grep enabled")"]
    assert [ -n "$(run lxc_exec "${containers_to_upgrade[4]}" "microovn status | grep -ozE 'microovn-central-control-5[^-]*' | grep central")"]
    assert [ -n "$(run lxc_exec "${containers_to_upgrade[4]}" "snap services microovn | grep ovn-northd | grep enabled")"]
    assert [ -n "$(run lxc_exec "${containers_to_upgrade[5]}" "microovn status | grep -ozE 'microovn-central-control-6[^-]*' | grep central")"]
    assert [ -n "$(run lxc_exec "${containers_to_upgrade[5]}" "snap services microovn | grep ovn-northd | grep enabled")"]
}

central_register_test_functions


load ${ABS_TOP_TEST_DIRNAME}test_helper/bats/cluster.bats
