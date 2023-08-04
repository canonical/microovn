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
