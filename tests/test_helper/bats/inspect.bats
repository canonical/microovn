# This is a bash shell fragment -*- bash -*-

load "${ABS_TOP_TEST_DIRNAME}test_helper/setup_teardown/$(basename "${BATS_TEST_FILENAME//.bats/.bash}")"

setup() {
    load test_helper/common.bash
    load test_helper/lxd.bash
    load test_helper/microovn.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-support/load.bash
    load ${ABS_TOP_TEST_DIRNAME}../.bats/bats-assert/load.bash

    assert [ -n "$TEST_CONTAINERS" ]

    read -r INSPECT_LEADER _ _ INSPECT_NON_VOTER <<< "$TEST_CONTAINERS"
    INSPECT_ENV_CONTAINER=""
    INSPECT_UNEXPECTED_SERVICE_CONTAINER=""
    INSPECT_STOPPED_CONTAINER=""
    INSPECT_STOPPED_SERVICE_CONTAINER=""
    INSPECT_NETWORK_SWITCH=""
    INSPECT_NETWORK_DHCP_UUID=""
}

teardown() {
    if [ -n "$INSPECT_ENV_CONTAINER" ]; then
        lxc_exec "$INSPECT_ENV_CONTAINER" \
            "sed -i '/^MICROOVN_INSPECT_TEST=/d' /var/snap/microovn/common/data/env/ovn.env" || true
    fi

    if [ -n "$INSPECT_STOPPED_SERVICE_CONTAINER" ]; then
        lxc_exec "$INSPECT_STOPPED_SERVICE_CONTAINER" \
            "snap start microovn.switch" || true
    fi

    if [ -n "$INSPECT_UNEXPECTED_SERVICE_CONTAINER" ]; then
        lxc_exec "$INSPECT_UNEXPECTED_SERVICE_CONTAINER" \
            "microovn enable switch" || true
    fi

    if [ -n "$INSPECT_STOPPED_CONTAINER" ]; then
        lxc start "$INSPECT_STOPPED_CONTAINER" || true
        wait_containers_ready "$INSPECT_STOPPED_CONTAINER" || true
    fi

    if [ -n "$INSPECT_NETWORK_SWITCH" ]; then
        lxc_exec "$INSPECT_LEADER" \
            "microovn.ovn-nbctl --if-exists ls-del '$INSPECT_NETWORK_SWITCH'" || true
    fi

    if [ -n "$INSPECT_NETWORK_DHCP_UUID" ]; then
        lxc_exec "$INSPECT_LEADER" \
            "microovn.ovn-nbctl --if-exists destroy DHCP_Options '$INSPECT_NETWORK_DHCP_UUID'" || true
    fi

    print_diagnostics_on_failure $TEST_CONTAINERS
}

@test "Inspect reports a healthy deployment without changing status" {
    local status_before
    local status_after
    status_before=$(lxc_exec "$INSPECT_LEADER" "microovn status")

    run lxc_exec "$INSPECT_LEADER" "microovn inspect"
    assert_success
    assert_output -p "Execution: node=$INSPECT_LEADER role=voter scope=cluster authoritative=true"
    assert_output -p "Database:"
    assert_output -p "Summary: PASS"

    run lxc_exec "$INSPECT_LEADER" "microovn inspect --verbose"
    assert_success
    assert_output -p "[PASS]"

    status_after=$(lxc_exec "$INSPECT_LEADER" "microovn status")
    assert_equal "$status_after" "$status_before"
}

@test "Inspect produces a complete versioned JSON report" {
    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .schema_version == 1 and
            .execution_context.authoritative == true and
            .execution_context.scope == \"cluster\" and
            .summary.status == \"PASS\" and
            .summary.counts.warning == 0 and
            .summary.counts.fail == 0 and
            .summary.counts.unknown == 0 and
            (.database_summary | has(\"northbound\") and has(\"southbound\") and has(\"communication\")) and
            (.results | length > 0)
        '"
    assert_success
}

@test "Inspect rejects incompatible output flags with exit code 2" {
    run lxc_exec "$INSPECT_LEADER" "microovn inspect --verbose --format=json"
    assert_equal "$status" 2
    assert_output -p "--verbose and --format=json cannot be used together"
}

@test "Inspect reports a stopped configured service" {
    INSPECT_STOPPED_SERVICE_CONTAINER="$INSPECT_NON_VOTER"
    run lxc_exec "$INSPECT_STOPPED_SERVICE_CONTAINER" "snap stop microovn.switch"
    assert_success

    run lxc_exec "$INSPECT_LEADER" "microovn inspect --format=json"
    assert_equal "$status" 1

    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .summary.status == \"FAIL\" and
            .summary.counts.fail > 0 and
            any(.results[]; .status == \"FAIL\") and
            (tostring | contains(\"$INSPECT_STOPPED_SERVICE_CONTAINER\")) and
            (tostring | contains(\"microovn.switch\"))
        '"
    assert_success
}

@test "Inspect warns about an unexpected active service" {
    INSPECT_UNEXPECTED_SERVICE_CONTAINER="$INSPECT_LEADER"
    run lxc_exec "$INSPECT_UNEXPECTED_SERVICE_CONTAINER" "microovn disable switch"
    assert_success
    run lxc_exec "$INSPECT_UNEXPECTED_SERVICE_CONTAINER" "snap start microovn.switch"
    assert_success

    run lxc_exec "$INSPECT_LEADER" "microovn inspect --format=json"
    assert_equal "$status" 1

    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .summary.counts.warning > 0 and
            any(.results[]; .id == \"service-runtime-drift\" and .status == \"WARNING\") and
            (tostring | contains(\"$INSPECT_UNEXPECTED_SERVICE_CONTAINER\")) and
            (tostring | contains(\"microovn.switch\"))
        '"
    assert_success
}

@test "Inspect reports environment drift without exposing values" {
    local marker="inspect-value-must-not-be-reported"
    INSPECT_ENV_CONTAINER="$INSPECT_NON_VOTER"
    lxc_exec "$INSPECT_ENV_CONTAINER" \
        "printf '%s\n' 'MICROOVN_INSPECT_TEST=\"$marker\"' >> /var/snap/microovn/common/data/env/ovn.env"

    run lxc_exec "$INSPECT_LEADER" "microovn inspect --format=json"
    assert_equal "$status" 1
    refute_output -p "$marker"

    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .summary.counts.warning > 0 and
            any(.results[]; .category == \"environment\" and .status == \"WARNING\") and
            (tostring | contains(\"MICROOVN_INSPECT_TEST\")) and
            (tostring | contains(\"$marker\") | not)
        '"
    assert_success
}

@test "Inspect reports and recovers from a missing DHCP metadata route" {
    INSPECT_NETWORK_SWITCH="inspect-metadata-route"
    local port="inspect-metadata-port"
    local cidr="10.250.0.0/24"

    run lxc_exec "$INSPECT_LEADER" \
        "microovn.ovn-nbctl ls-add '$INSPECT_NETWORK_SWITCH'"
    assert_success

    INSPECT_NETWORK_DHCP_UUID=$(lxc_exec "$INSPECT_LEADER" \
        "microovn.ovn-nbctl create DHCP_Options cidr='$cidr'")
    assert [ -n "$INSPECT_NETWORK_DHCP_UUID" ]

    run lxc_exec "$INSPECT_LEADER" \
        "microovn.ovn-nbctl lsp-add '$INSPECT_NETWORK_SWITCH' '$port'"
    assert_success
    run lxc_exec "$INSPECT_LEADER" \
        "microovn.ovn-nbctl set Logical_Switch_Port '$port' dhcpv4_options='$INSPECT_NETWORK_DHCP_UUID'"
    assert_success

    run lxc_exec "$INSPECT_LEADER" "microovn inspect --format=json"
    assert_equal "$status" 1

    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .summary.status == \"FAIL\" and
            any(.results[];
                .id == \"dhcp-metadata-route\" and
                .status == \"FAIL\" and
                any(.details[];
                    .data.uuid == \"$INSPECT_NETWORK_DHCP_UUID\" and
                    .data.cidr == \"$cidr\" and
                    .data.ports == \"$port\"
                )
            )
        '"
    assert_success

    run lxc_exec "$INSPECT_LEADER" \
        "microovn.ovn-nbctl set DHCP_Options '$INSPECT_NETWORK_DHCP_UUID' options:classless_static_route='{169.254.169.254/32,10.250.0.2}'"
    assert_success

    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .summary.status == \"PASS\" and
            any(.results[];
                .id == \"network\" and
                .status == \"PASS\" and
                .summary == \"DHCP metadata routes are configured\"
            )
        '"
    assert_success
}

@test "Inspect preserves partial results when a peer is unavailable" {
    INSPECT_STOPPED_CONTAINER="$INSPECT_NON_VOTER"
    run lxc stop --force "$INSPECT_STOPPED_CONTAINER"
    assert_success

    run lxc_exec "$INSPECT_LEADER" "microovn inspect --format=json"
    assert_equal "$status" 1

    run lxc_exec "$INSPECT_LEADER" \
        "microovn inspect --format=json | jq -e '
            .summary.status == \"UNKNOWN\" and
            .summary.counts.pass > 0 and
            .summary.counts.unknown > 0 and
            any(.results[]; .status == \"UNKNOWN\") and
            (tostring | contains(\"$INSPECT_STOPPED_CONTAINER\"))
        '"
    assert_success
}

@test "Inspect degrades to local evidence on a non-voting member" {
    run lxc_exec "$INSPECT_NON_VOTER" "microovn inspect --format=json"
    assert_equal "$status" 1

    run lxc_exec "$INSPECT_NON_VOTER" \
        "microovn inspect --format=json | jq -e '
            .execution_context.authoritative == false and
            .execution_context.scope == \"local\" and
            (.execution_context.member_role == \"standby\" or .execution_context.member_role == \"spare\") and
            .summary.status != \"PASS\" and
            .summary.counts.unknown > 0
        '"
    assert_success
}
