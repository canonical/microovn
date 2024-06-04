# get_current_ovsdb_schema_version CONTAINER NBSB
#
# Use CONTAINER to retrieve currently active OVN Northbound or Southbound
# database schema. This function expect that MicroOVN is already installed
# and bootstrapped in the CONTAINER.
# Valid values for NBSB argument are either lowercase "sb" or "nb".
#
# NOTE: This function can be run only on containers that run "central" services
#       as it requires local unix socket for the database
function get_current_ovsdb_schema_version() {
    local container=$1; shift
    local nbsb=$1; shift

    local schema_name=""
    local connection_string=""

    if [ $nbsb == "nb" ]; then
        schema_name="OVN_Northbound"
        connection_string="unix:/var/snap/microovn/common/run/ovn/ovnnb_db.sock"
        elif [ $nbsb == "sb" ]; then
            schema_name="OVN_Southbound"
            connection_string="unix:/var/snap/microovn/common/run/ovn/ovnsb_db.sock"
        else
            echo "# Unknown database type '$nbsb'. Valid values: 'nb', 'sb'"
            return 1
    fi

    local pki_dir="/var/snap/microovn/common/data/pki/"
    local cert="$pki_dir/client-cert.pem"
    local key="$pki_dir/client-privkey.pem"
    local ca_cert="$pki_dir/cacert.pem"

    lxc_exec "$container" "microovn.ovsdb-client -t 10 -c $cert -p $key -C $ca_cert \
                           get-schema-version $connection_string $schema_name"
}

# get_ovsdb_schema_version_from_snap CONTAINER SNAP_PATH NBSB
#
# This function returns Northbound or Southbound database schema version that's
# packed with the MicroOVN snap package located at SNAP_PATH.
#
# This function does not install the snap in the CONTAINER. It only unpacks it and uses
# OVS tools to determine the schema version.
# This function expects that MicroOVN is already installed in the CONTAINER so that its
# tools can be used to determine schema version.
# Valid values for NBSB argument are either lowercase "sb" or "nb".
function get_ovsdb_schema_version_from_snap(){
    local container=$1; shift
    local snap_path=$1; shift
    local nbsb=$1; shift

    local unsquash_path="/root/snap/microovn/common/microovn-squashfs-root"
    local container_snap_path="/tmp/microovn-to-unsquash.snap"
    local schema_path=""

    if [ $nbsb == "nb" ]; then
        schema_path="$unsquash_path/share/ovn/ovn-nb.ovsschema"
        elif [ $nbsb == "sb" ]; then
        schema_path="$unsquash_path/share/ovn/ovn-sb.ovsschema"
        else
            echo "# Unknown database type '$nbsb'. Valid values: 'nb', 'sb'"
            return 1
    fi


    lxc_file_push "$snap_path" "$container$container_snap_path"
    lxc_exec "$container" "unsquashfs -q -n -d $unsquash_path $container_snap_path"

    local schema_version=""
    schema_version=$(lxc_exec "$container" "microovn.ovsdb-tool schema-version $schema_path")


    lxc_exec "$container" "rm -rf $unsquash_path"
    lxc_exec "$container" "rm $container_snap_path"

    if [ -z "$schema_version" ]; then
        return 1
    fi

    echo "$schema_version"
}
