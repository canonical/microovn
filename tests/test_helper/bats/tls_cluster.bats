# This is a bash shell fragment -*- bash -*-
load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load ${ABS_TOP_TEST_DIRNAME}test_helper/common.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/lxd.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/tls.bash
    load ${ABS_TOP_TEST_DIRNAME}test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    # Ensure TEST_CONTAINERS is populated, otherwise the tests below will
    # provide false positive results.
    assert [ -n "$TEST_CONTAINERS" ]
    assert [ -n "$CENTRAL_CONTAINERS" ]
    assert [ -n "$CHASSIS_CONTAINERS" ]
}

@test "OVN central services have enabled TLS" {
    # Ensure that OVN service are listening on ports with TLS enabled
    local ports="6641 6642 6643 6644"

    for container in $CENTRAL_CONTAINERS; do
        local ip_addr=""
        ip_addr=$(microovn_get_cluster_address "$container")
        for port in $ports; do
            echo "# Checking port $port on $container"
            verify_service_cert "$container" "$ip_addr" "$port"
        done
    done
}

@test "Certificates files are valid certificates" {
    # Validate certificate files issued by MicroOVN

    for container in $CENTRAL_CONTAINERS; do
        verify_central_cert_files "$container"
    done

    for container in $CHASSIS_CONTAINERS; do
        verify_chassis_cert_files "$container"
    done
}

@test "List certificates on OVN Central node" {
    # Ensure that expected certificates are listed in the output of the 'microovn certificates list'
    # command.
    local container=""
    container=$(echo "$CENTRAL_CONTAINERS" | awk '{print $1;}')
    local expected_output='{
  "ca": "/var/snap/microovn/common/data/pki/cacert.pem",
  "ovnnb": {
    "cert": "/var/snap/microovn/common/data/pki/ovnnb-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/ovnnb-privkey.pem"
  },
  "ovnsb": {
    "cert": "/var/snap/microovn/common/data/pki/ovnsb-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/ovnsb-privkey.pem"
  },
  "ovn-northd": {
    "cert": "/var/snap/microovn/common/data/pki/ovn-northd-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/ovn-northd-privkey.pem"
  },
  "ovn-controller": {
    "cert": "/var/snap/microovn/common/data/pki/ovn-controller-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/ovn-controller-privkey.pem"
  },
  "client": {
    "cert": "/var/snap/microovn/common/data/pki/client-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/client-privkey.pem"
  }
}'
    run lxc_exec "$container" "microovn certificates list --format json | jq"
    assert_success
    assert_output "$expected_output"

}

@test "List certificates on OVN Chassis node" {
    # Ensure that expected certificates are listed in the output of the 'microovn certificates list'
    # command.
    local container=""
    container=$(echo "$CHASSIS_CONTAINERS" | awk '{print $1;}')
    local expected_output='{
  "ca": "/var/snap/microovn/common/data/pki/cacert.pem",
  "ovnnb": null,
  "ovnsb": null,
  "ovn-northd": null,
  "ovn-controller": {
    "cert": "/var/snap/microovn/common/data/pki/ovn-controller-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/ovn-controller-privkey.pem"
  },
  "client": {
    "cert": "/var/snap/microovn/common/data/pki/client-cert.pem",
    "key": "/var/snap/microovn/common/data/pki/client-privkey.pem"
  }
}'
    run lxc_exec "$container" "microovn certificates list --format json | jq"
    assert_success
    assert_output "$expected_output"

}

@test "Reissue individual certificates on OVN Central node" {
    # Ensure that MicroOVN is capable of individually re-issuing certificates used on OVN central nodes
    local container=""
    container=$(echo "$CENTRAL_CONTAINERS" | awk '{print $1;}')
    declare -A services=(\
        ["client"]=$CLIENT_CERT_PATH\
        ["ovnnb"]=$OVN_NB_CERT_PATH\
        ["ovnsb"]=$OVN_SB_CERT_PATH\
        ["ovn-controller"]=$CONTROLLER_CERT_PATH\
        ["ovn-northd"]=$NORTHD_CERT_PATH\
    )

    for service in "${!services[@]}"; do
        echo "# ($container) Reissuing certificate for $service"
        local cert_path="${services[$service]}"
        local old_hash=""
        local new_hash=""

        old_hash=$(get_cert_fingerprint "$container" "$cert_path")
        run reissue_certificate "$container" "$service"
        new_hash=$(get_cert_fingerprint "$container" "$cert_path")

        assert_success
        assert [ "$old_hash" != "$new_hash" ]
    done

    verify_central_cert_files "$container"
}

@test "Reissue individual certificates on OVN Chassis node" {
    # Ensure that MicroOVN is capable of individually re-issuing certificates used on OVN chassis nodes
    local container=""
    container=$(echo "$CHASSIS_CONTAINERS" | awk '{print $1;}')
    declare -A enabled_services=(\
        ["client"]=$CLIENT_CERT_PATH\
        ["ovn-controller"]=$CONTROLLER_CERT_PATH\
    )

    declare -A disabled_services=(\
        ["ovnnb"]=$OVN_NB_CERT_PATH\
        ["ovnsb"]=$OVN_SB_CERT_PATH\
        ["ovn-northd"]=$NORTHD_CERT_PATH\
    )

    for service in "${!enabled_services[@]}"; do
        # Ensure that certificates for enabled services can be refreshed
        echo "# ($container) Reissuing certificate for $service"
        local cert_path="${enabled_services[$service]}"
        local old_hash=""
        local new_hash=""

        old_hash=$(get_cert_fingerprint "$container" "$cert_path")
        run reissue_certificate "$container" "$service"
        new_hash=$(get_cert_fingerprint "$container" "$cert_path")

        assert_success
        assert [ "$old_hash" != "$new_hash" ]
    done


    for service in "${!disabled_services[@]}"; do
        # Ensure that certificates for disabled services can not be refreshed
        echo "# ($container) Attempting to reissue certificate for $service. Expecting failure"
        local cert_path="${disabled_services[$service]}"

        run lxc_exec "$container" "ls $cert_path"
        assert_failure

        run reissue_certificate "$container" "$service"
        assert_failure
        assert_output -p "Can't issue certificate for service '$service'. Service is not enabled on this member."

        run lxc_exec "$container" "ls $cert_path"
        assert_failure
    done

    verify_chassis_cert_files "$container"
}

@test "Reissue all certificates on OVN Central node" {
    # Ensure that MicroOVN can reissue certificate using magic argument 'all'
    local container=""
    container=$(echo "$CENTRAL_CONTAINERS" | awk '{print $1;}')
    declare -A services=(\
        ["client"]=$CLIENT_CERT_PATH\
        ["ovnnb"]=$OVN_NB_CERT_PATH\
        ["ovnsb"]=$OVN_SB_CERT_PATH\
        ["ovn-controller"]=$CONTROLLER_CERT_PATH\
        ["ovn-northd"]=$NORTHD_CERT_PATH\
    )
    declare -A old_hashes=()
    local old_ca_hash=""
    local new_ca_hash=""
    old_ca_hash=$(get_cert_fingerprint "$container" "$CA_CERT_PATH")

    # Collect original certificate fingerprints
    for service in "${!services[@]}"; do
        local cert_path="${services[$service]}"
        old_hashes["$service"]=$(get_cert_fingerprint "$container" "$cert_path")
    done

    run lxc_exec "$container" "microovn certificates reissue all"
    assert_success

    # Verify that certificates have new fingerprints
    for service in "${!services[@]}"; do
        local cert_path="${services[$service]}"
        local new_hash=""
        local old_hash=""

        new_hash=$(get_cert_fingerprint "$container" "$cert_path")
        old_hash="${old_hashes[$service]}"

        assert [ "$old_hash" != "$new_hash" ]
    done

    # Verify that CA certificate itself did not change
    new_ca_hash=$(get_cert_fingerprint "$container" "$CA_CERT_PATH")
    assert [ "$old_ca_hash" == "$new_ca_hash" ]

    verify_central_cert_files "$container"
}

@test "Reissue all certificates on OVN Chassis node" {
    # Ensure that MicroOVN does not issue certificates for disabled services when using magic argument 'all'
    # Remaining functionality of 'microovn certificates reissue all' is tested in "central" node test.
    local container=""
    container=$(echo "$CHASSIS_CONTAINERS" | awk '{print $1;}')
    declare -A disabled_services=(\
        ["ovnnb"]=$OVN_NB_CERT_PATH\
        ["ovnsb"]=$OVN_SB_CERT_PATH\
        ["ovn-northd"]=$NORTHD_CERT_PATH\
    )

    run lxc_exec "$container" "microovn certificates reissue all"
    assert_success


    for service in "${!disabled_services[@]}"; do
        # Ensure that certificates for disabled services were not created
        local cert_path="${disabled_services[$service]}"

        run lxc_exec "$container" "ls $cert_path"
        assert_failure
    done

    verify_chassis_cert_files "$container"
}

@test "Regenerate CA" {
    # Test recreation of the entire PKI. New CA should be created and then used to
    # reissue all server/client certificates
    local container=""
    container=$(echo "$TEST_CONTAINERS" | awk '{print $1;}')
    local old_ca_hash=""

    # Sample old CA certificate fingerprint from random host
    old_ca_hash=$(get_cert_fingerprint "$container" "$CA_CERT_PATH")

    # Trigger PKI regeneration
    run lxc_exec "$container" "microovn certificates regenerate-ca"
    assert_success

    # Sample new CA certificate fingerprint from random host
    new_ca_hash=$(get_cert_fingerprint "$container" "$CA_CERT_PATH")

    # Ensure that all members have new CA
    for container in $TEST_CONTAINERS; do
        local local_ca_hash=""
        local_ca_hash=$(get_cert_fingerprint "$container" "$CA_CERT_PATH")

        assert [ "$local_ca_hash" == "$new_ca_hash" ]
    done

    # Ensure that OVN Central nodes have certificates signed by new CA
    for container in $CENTRAL_CONTAINERS; do
        verify_central_cert_files "$container"
    done

    # Ensure that OVN Chassis nodes have certificates signed by new CA
    for container in $CHASSIS_CONTAINERS; do
        verify_chassis_cert_files "$container"
    done
}
