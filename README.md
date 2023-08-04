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

The value of `<member_name>` is taken from the **Name** column in the output
of the `microovn cluster list` command.

Any chassis components (`ovn-controller` and `ovs-vswitchd`) running on the
member will first be stopped and disabled (prevented from starting). For a
member with central components present (`microovn.central`), the Northbound and
Southbound databases will be gracefully removed.

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
