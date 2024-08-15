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

central_tests() {
    read -r -a containers_to_upgrade <<< "$TEST_CONTAINERS"
    run lxc_exec "${containers_to_upgrade[0]}" "microovn disable central"
    assert_output -p "Service central disabled"
    run lxc_exec "${containers_to_upgrade[1]}" "microovn disable central"
    assert_output -p "Service central disabled"
    run lxc_exec "${containers_to_upgrade[3]}" "microovn enable central"
    assert_output -p "Service central enabled"
    run lxc_exec "${containers_to_upgrade[2]}" "microovn disable central"
    assert_output -p "Service central disabled"
    run lxc_exec "${containers_to_upgrade[4]}" "microovn enable central"
    assert_output -p "Service central enabled"
    run lxc_exec "${containers_to_upgrade[5]}" "microovn enable central"
    assert_output -p "Service central enabled"
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
