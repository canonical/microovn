# MicroOVN

MicroOVN is snap-deployed OVN with built-in clustering.

## Installation

MicroOVN is distributed as a snap and can be installed with:

```shell
snap install microovn
```

More installation information can be found on its [Snap Store page](https://snapcraft.io/microovn).

## Single node deployment

This is the simplest way to deploy OVN, with no redundancy. After `MicroOVN`
snap installation simply run:

```shell
microovn cluster bootstrap
```

and that's it. Your OVN deployment is ready for you. You can interact with it
using usual OVN tools prefaced with `microovn.`. For example, to show
contents of the OVN Southbound database, you can run:

```shell
microovn.ovn-sbctl show
```

## 3-node deployment

Following steps will demonstrate how to quickly deploy MicroOVN cluster on
three nodes. For this exercise you'll need three standalone virtual or physical
machines, all with `microovn` snap installed.

Start by initializing MicroOVN cluster and generating access tokens
for expected members.

```shell
## On node-1

# Initialize cluster
microovn cluster bootstrap

# Generate access token for node-2
microovn cluster add node-2
# Outputs: eyJuYW1lIjoibm9kZS0yIiwic2VjcmV0IjoiMzBlM...

# Generate access token for node-3
microovn cluster add node-3
# Outputs: eyJuYW1lIjoibm9kZS0zIiwic2VjcmV0IjoiZmZhY...
```

Then join the cluster with other two members using generated access tokens.

```shell
## On node-2

# Join cluster with access token for node-2
microovn cluster join eyJuYW1lIjoibm9kZS0yIiwic2VjcmV0IjoiMzBlM...
```

```shell
## On node-3

# Join cluster with access token for node-3
microovn cluster join eyJuYW1lIjoibm9kZS0zIiwic2VjcmV0IjoiZmZhY...
```

Now all three members are joined in a cluster, and you can interact with it
using common OVN tools, prefaced with `microovn.`, on any of the nodes.
For example:

```shell
# Show overview of OVN Southbound database
microovn.ovn-sbctl show

# Or check out the Northbound cluster status
microovn.ovn-appctl -t /var/snap/microovn/common/run/central/ovnnb_db.ctl cluster/status OVN_Northbound
```

## Downscaling

### Impact

Downscaling can have an adverse effect on the availability and resiliency of
the cluster, especially when a member is being removed that runs an OVN central
service (OVN SB, OVN NB, OVN Northd).

OVN uses the [Raft consensus algorithm](https://raft.github.io) for cluster
management, which has a fault tolerance of up to `(N-1)/2` members. This means
that fault resiliency will be lost if a three-node cluster is reduced to two
nodes.

### Monitoring

You can watch logs on the departing member for indications of removal failures
with:

    snap logs -f microovn.daemon

Any issues that arise during the removal process will need to be resolved
manually.

### Remove a cluster member

To remove a cluster member:

    microovn cluster remove <member_name>

The value of <member_name> is taken from the **Name** column in the output
of the `microovn cluster list` command.

Any chassis components (`ovn-controller` and `ovs-vswitchd`) running on the
member will first be stopped and disabled (prevented from starting). For a
member with central components present (`ovn-southd` and `ovn-northd`), the
Northbound and Southbound databases will be gracefully removed.

### Verification

Upon removal, check the state of OVN services to ensure that the member was
properly removed.

```
# Check status of OVN SB cluster
microovn.ovn-appctl -t /var/snap/microovn/common/run/central/ovnsb_db.ctl cluster/status OVN_Southbound

# Check status of OVN NB cluster
microovn.ovn-appctl -t /var/snap/microovn/common/run/central/ovnnb_db.ctl cluster/status OVN_Northbound

# Check registered chassis
microovn.ovn-sbctl show
```

### Data preservation

MicroOVN will back up selected data directories into the timestamped location
`/var/snap/microovn/common/backup_<timestamp>/`. These backups will include:

* logs
* OVN database files
* OVS database file
* issued certificates and keys

## TLS

Starting with snap revision `111`, new deployments of MicroOVN use TLS
encryption by default. A self-signed CA certificate is used to issue
certificates to all OVN services that require it. They provide authentication
and encryption for OVSDB communication. The CA certificate is generated during
cluster initialisation (`cluster bootstrap` command).

In the current implementation, self-provisioned certificates are the only mode
available. Future releases may include support for externally provided
certificates.

> **Warning**
>
> The certificate and private key generated for the self-provisioned CA are
> currently stored unencrypted in the database on every cluster member. If an
> attacker gains access to any cluster member, they can use the CA to issue
> valid certificates that will be accepted by other cluster members.

### Certificates CLI

MicroOVN exposes a few commands for basic interaction with TLS certificates.

#### List certificates

To list currently used certificates:

    microovn certificates list

Example output:

```
[OVN CA]
/var/snap/microovn/common/data/pki/cacert.pem (OK: Present)

[OVN Northbound Service]
/var/snap/microovn/common/data/pki/ovnnb-cert.pem (OK: Present)
/var/snap/microovn/common/data/pki/ovnnb-privkey.pem (OK: Present)

[OVN Southbound Service]
/var/snap/microovn/common/data/pki/ovnsb-cert.pem (OK: Present)
/var/snap/microovn/common/data/pki/ovnsb-privkey.pem (OK: Present)

[OVN Northd Service]
/var/snap/microovn/common/data/pki/ovn-northd-cert.pem (OK: Present)
/var/snap/microovn/common/data/pki/ovn-northd-privkey.pem (OK: Present)

[OVN Chassis Service]
/var/snap/microovn/common/data/pki/ovn-controller-cert.pem (OK: Present)
/var/snap/microovn/common/data/pki/ovn-controller-privkey.pem (OK: Present)
```

This command does not perform any certificate validation, it only ensures that
if a service is available on the node, the file that should contain a
certificate is in place.

#### Re-issue certificates

The `certificates reissue` command is used to interact with OVN services on the
local host; it does not affect peer cluster members.

> **Important**
>
> Services must be running in order to be affected by the `certificates
> reissue` command.

To re-issue a certificate for a single service:

    microovn certificates reissue <ovn_service_name>

To re-issue certificates for all services, the `all` argument is supported:

    microovn certificates reissue all

Valid service names can be discovered with the `--help` option:

    microovn certificates reissue --help

#### Regenerate PKI for the cluster

The `certificates regenerate-ca` command is used to issue a new CA certificate
and new certificates for every OVN service in the cluster:

    microovn certificates regenerate-ca

This command replaces the current CA certificate and notifies all cluster
members to re-issue certificates for all their services. The command's output
will include evidence of successfully issued certificates for each cluster
member.

> **Warning**
>
> A new certificate must be issued successfully for every service on every
> member. Any failure will result in subsequent communication errors for that
> service within the cluster.

### Certificate lifecycle

Certificates that are automatically provisioned by MicroOVN have the following
lifespans:

* CA certificate: 10 years
* OVN service certificate: 2 years

MicroOVN runs daily checks for certificate lifespan validity. When a
certificate is within 10 days of expiration, it will be automatically renewed.

### Upgrade from plaintext to TLS

Plaintext communication is used when MicroOVN is initially deployed with a snap
revision of less than `111`, and there's no way to automatically convert to
encrypted communication. The following manual steps are needed to upgrade from
plaintext to TLS:

* ensure that all MicroOVN snaps in the cluster are upgraded to, at least,
revision `111`
* run `microovn certificates regenerate-ca` on one of the cluster members
* run `sudo snap restart microovn.daemon` on **all** cluster members

Once this is done, OVN services throughout the cluster will start listening on
TLS-secured ports.

## Common issues

This section contains some well known or expected issues that you can encounter.

### I'm getting `failed to load certificates` error

If you run commands like `microovn.ovn-sbctl` and you get complains about
missing certificates, while rest of the command seems to work fine.

Example:
```
root@microovn-0:~# microovn.ovn-sbctl show
2023-06-14T15:09:31Z|00001|stream_ssl|ERR|SSL_use_certificate_file: error:80000002:system library::No such file or directory
2023-06-14T15:09:31Z|00002|stream_ssl|ERR|SSL_use_PrivateKey_file: error:10080002:BIO routines::system lib
2023-06-14T15:09:31Z|00003|stream_ssl|ERR|failed to load client certificates from /var/snap/microovn/common/data/pki/cacert.pem: error:0A080002:SSL routines::system lib
Chassis microovn-0
    hostname: microovn-0
    Encap geneve
        ip: "10.5.3.129"
        options: {csum="true"}
```

This likely means that your MicroOVN snap got upgraded to a version that
supports TLS, but it requires some manual upgrade steps. Please see
[TLS upgrade guide](#upgrade-from-plaintext-to-tls)
