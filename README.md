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