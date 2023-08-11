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
    local family=${1:-inet}; shift

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
