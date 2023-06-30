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

> **Warning**
>
> Be aware of effect of downscaling on availability and resiliency of your
> cluster, especially when you are removing members that run OVN central
> service (OVN SB, OVN NB, OVN Northd). OVN uses
> [Raft consensus algorithm](https://raft.github.io) for cluster management,
> which has fault tolerance of up to `(N-1)/2` members. It means that if you
> scale back 3 node cluster to just 2 nodes, you lose any fault resiliency.

Removing MicroOVN cluster member is as easy as running

```shell
microovn cluster remove <member_name>
```

MicroOVN will stop and disable any chassis components running on the member
(`ovn-controller` and `ovs-vswitchd`), before attempting to gracefully leave
any OVN database clusters for members with central components present. You can
watch logs on departing member for any indications of removal failures with:

```shell
snap logs -f microovn.daemon
```

If there are any issues during the removal process, you will have to take
manual steps to fix the OVN cluster.

After the removal, you can also check the state of OVN services, to make sure
that the member was properly removed.

```shell
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

* Logs
* OVN database files
* OVS database file
* Issued certificates and keys

## TLS encryption

MicroOVN enables SSL/TLS in OVN by default. It uses self-signed CA certificate
generated during bootstrap process to issue certificates for all the OVN
services that require it.

> **Warning**
>
> Certificate and private key generated for the self-provisioned CA by MicroOVN
> are, in the current implementation, stored unencrypted in database on every
> cluster member. This means that if attackers gain access to even one cluster
> member, they can use the CA to issue valid certificates accepted by other
> cluster members.

In the current implementation, self-provisioned certificates are the only mode
available, and they provide authentication and encryption for the OVSDB
communication. For the future, we are considering support for externally
provided certificates.


### Certificates CLI

MicroOVN exposes few commands for basic interaction with self-provisioned
certificates

#### List certificates

To show list of currently used certificates, run:

```shell
microovn certificates list
```

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
if a service is available on the node, the file that should contain certificate
is in place.

#### Re-issue certificates

To manually issue new certificate for a specific OVN service, run:

```shell
microovn certificates reissue <ovn_service_name>
```

For list of all valid service names, see output of

```shell
microovn certificates reissue --help
```

This command has only local scope and will issue new certificate only for an
OVN service running on the local host. Other members of the MicroOVN cluster
are not affected by this command. Additionally, the certificate will be issued
only if the service is enabled on the host.

Aside from specific OVN service name, this command also accepts argument `all`,
which results in issuing new certificate for every OVN service that's enabled
on the host.

#### Regenerate PKI completely

To issue new CA certificate and new certificate for every OVN service
in the cluster, run:

```shell
microovn certificates regenerate-ca
```

This command replaces current, shared, CA certificate and notifies all MicroOVN
cluster members to issue new certificates for all their OVN services. Output of
this command will contain report of successfully issued certificates on each
cluster member. Make sure that all services successfully received new
certificates, as the old certificates will no longer be accepted and services
that will keep using them won't be able to communicate with the rest of the
cluster.

### Certificates' lifecycle

Certificates that are automatically provisioned by MicroOVN have following
lifespans:

* CA certificate: 10 years
* OVN service certificate: 2 years

MicroOVN will run daily check on the remaining validity of all certificates.
When some of the certificates approach within 10 days of not being valid, they
will be automatically renewed

### Upgrade from plaintext to TLS

All new MicroOVN deployments have TLS enabled by default and there's no
supported way to revert back to the plaintext communication. However,
MicroOVN is not capable of transitioning existing deployments, that already use
plain text, to TLS automatically. Any deployments of a snap revision below `111`
will keep using plaintext even after the upgrade until following manual upgrade
steps are taken:

* Make sure that all MicroOVN snaps in your cluster are upgraded to, at least,
revision `111`
* Run `microovn certificates regenerate-ca` on one of the cluster members
* Run `snap restart microovn.daemon` on **all** cluster members

After these steps, OVN services in your cluster should start listening on ports,
using TLS certificates.

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