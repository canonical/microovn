WAIT_TIMEOUT=30

function container_name() {
    local container=$1; shift
    NAME_PREFIX=${NAME_PREFIX:-microovn}
    echo "${NAME_PREFIX}-${container}"
}

function container_names() {
    # shellcheck disable=SC2155
    local source_file_name=$(basename "$1" | cut -f1 -d\.); shift
    local nr_containers=$1; shift

    source_file_name=${source_file_name//_/-}

    for ((i=1; i <= nr_containers; i++)); do
        printf '%s ' "$(container_name "${source_file_name}-${i}")"
    done
}

function container_get_default_ip() {
    local container=$1; shift
    local family=${1:-inet}

    local dev
    dev=$(lxc_exec "$container" "ip route show default|awk '/dev/{print\$5}'")
    lxc_exec "$container" \
        "ip a show dev $dev | awk '/$family .*global/{print\$2}'|cut -f1 -d/"
}

function test_is_ipv6_test() {
    [[ "$BATS_TEST_FILENAME" == *"ipv6"* ]]
}

function test_ipv6_addr() {
    local addr=$1; shift

    [[ $addr == *":"* ]]
}

# print_address ADDRESS
#
# Prints ADDRESS, encapsulating it in brackets (`[]`) if it appears to be an
# IPv6 address.
function print_address() {
    local addr=$1; shift

    printf "%s%s%s\n" \
        "$(test_ipv6_addr $addr && echo '[' || true)" \
        "$addr" \
        "$(test_ipv6_addr $addr && echo ']' || true)"
}

# wait_for_open_port CONTAINER PORT MAX_RETRY
#
# Return after specified PORT is open and listening in the CONTAINER.
#
# The port can be bound to any interface and it's a good way to test whether
# OVN services are up and listening. This function will retry for maximum of
# MAX_RETRY times, each time backing of for 1 second between attempts.
function wait_for_open_port() {
    local container=$1; shift
    local port=$1; shift
    local max_retry=$1; shift
    local attempt=1
    local success=0

    while ! lxc_exec "$container" "lsof -i:$port -sTCP:LISTEN"; do
        echo "# ($container) waiting for port $port to be opened ($attempt/$max_retry)"
        if [ $attempt -gt "$max_retry" ]; then
            echo "# ($container) Maximum retries reached"
            success=1
            break
        fi
        ((++attempt))
        sleep 1
    done

    return $success
}

# get_pid_start_time CONTAINER PID
#
# Print the unix timestamp for when PID in CONTAINER started.
function get_pid_start_time() {
    local container=$1; shift
    local pid=$1; shift

    lxc_exec "$container" "stat -c %Y /proc/${pid}"
}

# wait_until WAIT_COND [ WAIT_FAILED ]
#
# Execute WAIT_COND until it returns 0, waiting up to WAIT_TIMEOUT seconds.
#
# Execute WAIT_FAILED on failure, if provided.
wait_until() {
    local wait_cond=$1; shift
    local wait_failed=${1:-false}

    _log_wait() {
        local how_soon=$1; shift
        printf '%q: wait succeeded %q\n' "$wait_cond" $how_soon
    }

    if $wait_cond; then _log_wait immediately; return 0; fi
    sleep 0.1
    if $wait_cond; then _log_wait quickly; return 0; fi

    local d
    for d in $(seq 1 "$WAIT_TIMEOUT"); do
        sleep 1
        if $wait_cond; then _log_wait "after $d seconds"; return 0; fi
    done

    printf '%q: wait failed after %s seconds\n' "$wait_cond" $d
    $wait_failed
    return 1
}

# snap_print_base SNAP
#
# Print base for SNAP
function snap_print_base() {
    local snap=$1; shift

    snap info --verbose "$snap" | awk '/base/{print$2}'
}

# test_snap_is_stable_base BASE-SNAP
#
# Returns 0 if base snap has stable channel, 1 otherwise
function test_snap_is_stable_base() {
    local base_snap=$1; shift

    local version_info
    version_info=$(snap info "$base_snap" | awk '/\/stable/{print$2}')

    [ "$version_info" != "--" ]
}

# get_upgrade_test_version TEST_FILE_NAME TEST_PREFIX
#
# Parse test filename of an "upgrade" test to determine which version should the
# test upgrade from.
#
# MicroOVN upgrade tests need to define initial MicroOVN version to be deployed, so that
# the tests can verify if the upgrade is possible. Initial version is defined in filename.
#
# For example, by passing "upgrade_22.03.bats" TEST_FILE_NAME and "upgrade" TEST_PREFIX into
# this function, it returns "22.03/stable".
#
# If there's no version defined in the test filename, (e.g. TEST_FILE_NAME is "upgrade.bats"
# and TEST_PREFIX is "upgrade"), an empty string is returned.
#
# If filename does not match expected formats "<TEST_PREFIX>_<MAJOR_VERSION>.<MINOR_VERSION>.bats",
# or "<TEST_PREFIX>.bats", this function returns with error code.
function get_upgrade_test_version() {
    local test_name=$1; shift
    local test_prefix=$1; shift

    upgrade_from_version=""

    if [ "$test_name" != "${test_prefix}.bats" ]; then
        upgrade_from_version=$(sed -nr 's/^'"$test_prefix"'_([0-9]+\.[0-9]+)\.bats/\1/p' <<< "$test_name")

        if [ -z "$upgrade_from_version" ]; then
            echo "Failed to determine MicroOVN upgrade version for this test: '$test_name'." >&2
            echo "" >&2
            echo "Expected test name is '${test_prefix}_<major_version>.<minor_version>.bats'" >&2
            echo "where '<major_version>.<minor_version>' determine MicroOVN track from which" >&2
            echo "the test will be performed." >&2
            exit 1
        fi

        upgrade_from_version="${upgrade_from_version}/stable"
    fi
    echo "$upgrade_from_version"
}

# netns_add CONTAINER NAME
#
# Add netns named NAME in CONTAINER.
function netns_add() {
    local container=$1; shift
    local name=$1; shift

    lxc_exec "$container" "ip netns add $name"
}

# netns_delete CONTAINER NAME
#
# Delete netns named NAME in CONTAINER.
function netns_delete() {
    local container=$1; shift
    local name=$1; shift

    lxc_exec "$container" "ip netns delete $name"
}

# netns_ifadd CONTAINER NAME IFNAME LLADDR CIDR
#
# Move the device identified by IFNAME into netns NAME in CONTAINER, set
# Link-Layer Address to LLADDR, add CIDR and bring the interface up.
function netns_ifadd() {
    local container=$1; shift
    local name=$1; shift
    local ifname=$1; shift
    local lladdr=$1; shift
    local cidr=$1; shift

    lxc_exec "$container" \
        "ip link set netns $name dev $ifname"
    lxc_exec "$container" \
        "ip netns exec $name ip link set address $lladdr dev $ifname"
    lxc_exec "$container" \
        "ip netns exec $name ip address add $cidr dev $ifname"
    lxc_exec "$container" \
        "ip netns exec $name ip link set up dev $ifname"
}

function _netns_dst_base_name() {
    local dst=$1; shift
    local netns=$1

    echo "${netns:+$netns-}${dst//[\.:]/_}"
}

# ping_start CONTAINER DST [ NETNS ]
#
# Start ping to DST in CONTAINER in the background, optionally using network
# namespace NETNS.
#
# Output from ping and the PID will be recorded as files in '/tmp/' in
# CONTAINER, and the process and results can be reaped by a subsequent call to
# ``ping_reap``.
function ping_start() {
    local container=$1; shift
    local dst=$1; shift
    local netns=$1

    local base_filename
    base_filename=$(_netns_dst_base_name "$dst" "$netns")

    lxc_exec "$container" \
        "${netns:+ip netns exec $netns} \
         ping $dst > /tmp/${base_filename}.stdout & \
         echo \$! > /tmp/${base_filename}.pid"
}

# ping_reap CONTAINER DST [ NETNS ]
#
# Stop ping process previously started by a call to ``ping_start`` and print
# its recorded output.
function ping_reap() {
    local container=$1; shift
    local dst=$1; shift
    local netns=$1

    local base_filename
    base_filename=$(_netns_dst_base_name "$dst" "$netns")


    lxc_exec "$container" \
        "pid=\$(cat /tmp/${base_filename}.pid) && \
         kill -INT \$pid && \
         while kill -0 \$pid; do sleep 0.1;done && \
         cat /tmp/${base_filename}.stdout"
}

# collect_coverage CONTAINERS
#
# NOTE: Coverage data collection is skipped unless environment
#       variable MICROOVN_COVERAGE_ENABLED  is set to "yes".
#
# WARNING: This function restarts microovn.daemon service on each container in
#          CONTAINERS, to force 'microovnd' process to write out its coverage data.
#
# For each container in CONTAINERS, pull runtime coverage data gathered from MicroOVN
# daemon and client binaries. Coverage data is by default collected
# in ".coverage/<test_name>/<container_name>". This location can be controlled by
# environment variable MICROOVN_COVERAGE_DST
function collect_coverage() {
    local containers=$*; shift

    if [ "$MICROOVN_COVERAGE_ENABLED" != "yes" ]; then
        echo "# Skipping coverage data collection" >&3
        return 0
    fi

    local test_name=""
    test_name=$(basename "$BATS_TEST_FILENAME" | cut -f1 -d\.)

    local dst_prefix="$MICROOVN_COVERAGE_DST"
    if [ -z "$dst_prefix" ]; then
        dst_prefix=".coverage"
    fi

    local container
    for container in $containers; do
        echo "# ($container) Collecting coverage information" >&3

        lxc_exec "$container" "snap restart microovn.daemon"

        local output_dir="$dst_prefix/$test_name/$container"
        lxc_pull_dir "$container/var/snap/microovn/common/data/coverage" "$output_dir"
    done
}

# ping_packets_lost CONTAINER DST [ NETNS ]
#
# Stop ping process previously started by a call to ``ping_start`` and print
# how many packets were lost, if any.
function ping_packets_lost() {
    local container=$1; shift
    local dst=$1; shift
    local netns=$1

    local n_lost
    n_lost=$(ping_reap "$container" "$dst" "$netns" \
        | awk '/packets/{print$1-$4}')
    echo "$n_lost"
}

# install_apt_package CONTAINER PACKAGE
#
# install PACKAGE via apt in the CONTAINER
function install_apt_package() {
    local container=$1; shift
    local package=$1; shift

    lxc_exec "$container" "DEBIAN_FRONTEND=noninteractive apt install -yqq $package"
}
