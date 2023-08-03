_PKI_DIR="/var/snap/microovn/common/data/pki/"
CA_CERT_PATH="$_PKI_DIR""cacert.pem"
CLIENT_CERT_PATH="$_PKI_DIR""client-cert.pem"
CLIENT_KEY_PATH="$_PKI_DIR""client-privkey.pem"
CONTROLLER_CERT_PATH="$_PKI_DIR""ovn-controller-cert.pem"
CONTROLLER_KEY_PATH="$_PKI_DIR""ovn-controller-privkey.pem"
OVN_NB_CERT_PATH="$_PKI_DIR""ovnnb-cert.pem"
OVN_NB_KEY_PATH="$_PKI_DIR""ovnnb-privkey.pem"
OVN_SB_CERT_PATH="$_PKI_DIR""ovnsb-cert.pem"
OVN_SB_KEY_PATH="$_PKI_DIR""ovnsb-privkey.pem"
NORTHD_CERT_PATH="$_PKI_DIR""ovn-northd-cert.pem"
NORTHD_KEY_PATH="$_PKI_DIR""ovn-northd-privkey.pem"

# shellcheck disable=SC2034  # This variable is referenced and passed by name
declare -g -A OVN_CENTRAL_PKI=(\
    [$CLIENT_CERT_PATH]=$CLIENT_KEY_PATH\
    [$CONTROLLER_CERT_PATH]=$CONTROLLER_KEY_PATH\
    [$OVN_NB_CERT_PATH]=$OVN_NB_KEY_PATH\
    [$OVN_SB_CERT_PATH]=$OVN_SB_KEY_PATH\
    [$NORTHD_CERT_PATH]=$NORTHD_KEY_PATH\
)

# shellcheck disable=SC2034  # This variable is referenced and passed by name
declare -g -A OVN_CHASSIS_PKI=(\
     [$CLIENT_CERT_PATH]=$CLIENT_KEY_PATH\
     [$CONTROLLER_CERT_PATH]=$CONTROLLER_KEY_PATH\
 )

# verify_service_cert CONTAINER IP_ADDR PORT
#
# Ensure that service listening on IP_ADDR and PORT uses TLS and can be verified using
# CA certificate used by the MicroOVN on the specified CONTAINER.
function verify_service_cert() {
    local container=$1
    local ip_addr=$2
    local port=$3
    echo "# ($container) Checking TLS on $ip_addr:$port"
    lxc_exec "$container" "openssl s_client -CAfile $CA_CERT_PATH -cert $CLIENT_CERT_PATH \
        -key $CLIENT_KEY_PATH -verify_return_error -connect $ip_addr:$port <<< Q"
}

# _verify_cert_files CONTAINER CERTIFICATE_MAP
#
# Verify certificate files and keys specified by CERTIFICATE_MAP stored in the CONTAINER.
#
# CERTIFICATE_MAP should be passed as a variable name (not as a direct value) and the variable
# is expected to be an associative array that maps local paths to certificate to local paths of
# private keys. (see OVN_CENTRAL_PKI for example)
#
# Following checks are performed by this function:
#   * Ensure that the private key is used to sign the certificate
#   * Ensure that the certificate can be validated using CA certificate
function _verify_cert_files() {
    local container=$1
    local -n certificate_map=$2
    for cert in "${!certificate_map[@]}"; do
        local key="${certificate_map[$cert]}"

        echo "# ($container) Checking certificate $cert"
        # Check that private key matches the certificate
        cert_pubkey=$(lxc_exec "$container" "openssl x509 -noout -pubkey -in $cert")
        key_pubkey=$(lxc_exec "$container" "openssl ec -pubout -in $key")
        ## Check that relevant public key is found both in cert and in private key
        assert [ -n "$cert_pubkey" ]
        assert [ -n "$key_pubkey" ]
        ## Ensure that public key hashes match
        assert_equal "$(sha256sum "$cert_pubkey")" "$(sha256sum "$key_pubkey")"

        # Check that certificates match CA
        lxc_exec "$container" "openssl verify -CAfile $CA_CERT_PATH $cert"
    done
}

# verify_central_cert_files CONTAINER
#
# Verify set of certificate files that are expected to be present on the CONTAINER
# that runs "central" services.
#
# For more info about checks see _verify_cert_files
function verify_central_cert_files() {
    local container=$1
    _verify_cert_files "$1" OVN_CENTRAL_PKI
}

# verify_central_cert_files CONTAINER
#
# Verify set of certificate files that are expected to be present on the CONTAINER
# that runs "chassis" services.
#
# For more info about checks see _verify_cert_files
function verify_chassis_cert_files() {
    local container=$1
    _verify_cert_files "$1"  OVN_CHASSIS_PKI
}

# reissue_certificate CONTAINER SERVICE
#
# Issue new certificate for a specified SERVICE that runs in the CONTAINER.
#
# This functions is a wrapper for executing 'microovn certificates reissue'. See help
# output of that command for list of valid values for SERVICE argument.
function reissue_certificate() {
    # Issue new certificate for a specified OVN service in the container.
    local container=$1
    local service=$2
    lxc_exec "$container" "microovn certificates reissue $service"
}

# get_cert_fingerprint CONTAINER CERT_PATH
#
# Print fingerprint of a certificate inside a CONTAINER
function get_cert_fingerprint() {
    local container=$1
    local cert_path=$2
    local fingerprint=""

    fingerprint=$(lxc_exec "$container" "openssl x509 -in $cert_path -noout -fingerprint")
    assert [ -n "$fingerprint"]

    echo "$fingerprint"
}
