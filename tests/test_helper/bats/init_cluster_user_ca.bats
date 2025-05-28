# This is a bash shell fragment -*- bash -*-

load "test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load test_helper/tls.bash
    load ../.bats/bats-support/load.bash
    load ../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS are populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
}

init_cluster_user_ca_register_test_functions() {
    bats_test_function \
        --description "MicroOVN was bootstrapped with user-supplied CA" \
        -- cluster_bootstrapped_with_user_ca
}

cluster_bootstrapped_with_user_ca() {
    local container
    local uploaded_ca_hash
    local microovn_ca_hash

    uploaded_ca_hash=$(get_cert_fingerprint "$LEADER" "$USER_CA_CRT")
    assert [ -n "$uploaded_ca_hash" ]

    # Ensure that CA cert used by MicroOVN matches the one uploaded by the user
    for container in $TEST_CONTAINERS; do
        microovn_ca_hash=$(get_cert_fingerprint "$container" "$CA_CERT_PATH")
        assert [ -n "$microovn_ca_hash" ]
        assert_equal "$uploaded_ca_hash" "$microovn_ca_hash"
    done

    # Ensure that OVN Central nodes have certificates signed by the user-supplied CA
    for container in $CENTRAL_CONTAINERS; do
        verify_central_cert_files "$container"
    done

    # Ensure that OVN Chassis nodes have certificates signed by the user-supplied CA
    for container in $CENTRAL_CONTAINERS; do
        verify_chassis_cert_files "$container"
    done
}

init_cluster_user_ca_register_test_functions
