function container_name() {
    local container=$1; shift
    NAME_PREFIX=${NAME_PREFIX:-microovn}
    echo "${NAME_PREFIX}-${container}"
}

function container_names() {
    local source_file_name=$(basename $1 | cut -f1 -d\.); shift
    local nr_containers=$1; shift

    source_file_name=${source_file_name//_/-}

    for ((i=1; i <= nr_containers; i++)); do
        printf '%s ' "$(container_name "${source_file_name}-${i}")"
    done
}
