=======================
Security in MicroOVN
=======================

MicroOVN wraps OVN, OVS, and supporting services into an opinionated snap
deployment that prioritises security by default. This page gives a high-level
overview of the security posture and highlights areas that may need attention
depending on your deployment.

Security architecture
---------------------

MicroOVN runs as a `strictly confined snap`_, which limits filesystem and
network access to what is explicitly declared. All cluster-internal
communication (i.e. the MicroCluster REST API, OVN Northbound and Southbound
databases, and ``ovn-controller`` connections) is encrypted and authenticated
with mutual TLS. See :doc:`/reference/cryptography` for algorithm and
certificate details.

Cluster membership is controlled through single-use join tokens issued by
existing members. Once a node joins, it participates in a `dqlite`_ RAFT
cluster that stores MicroOVN's own state (configuration, certificate material,
service assignments). Each node independently manages OVN service certificates
using the shared Certificate Authority stored in dqlite.

.. _strictly confined snap: https://snapcraft.io/docs/snap-confinement
.. _dqlite: https://canonical.com/dqlite

Risks
-----

The following are known risks inherent to MicroOVN's design:

CA private key distribution
  The OVN Certificate Authority private key is stored in dqlite and replicated
  to every cluster member. Compromise of any single node gives an attacker the
  ability to issue certificates trusted by the entire cluster.

Data at rest
  Databases, certificates, and private keys on disk are stored unencrypted
  under ``/var/snap/microovn/common/data``, protected only by filesystem
  permissions (root-only). Full-disk encryption at the host level is
  recommended for sensitive environments.

Join token window
  Join tokens are single-use, but between issuance and consumption they grant
  full cluster membership to whoever presents them. Treat them as secrets and
  transfer them through secure channels.

BGP auto-configuration
  When BGP integration is enabled with the ``--asn`` option, the
  auto-configured BIRD daemon does not authenticate peers, validate routes via
  RPKI, or apply route filtering. See :doc:`/reference/hardening` for
  alternatives.

No OVSDB RBAC
  MicroOVN does not configure OVN's optional role-based access control on the
  Northbound or Southbound databases. Any client with valid TLS credentials has
  full read/write access.

Information security
--------------------

MicroOVN stores the following sensitive data:

* **OVN and OVS databases** (network topology, ACLs, logical flows), in OVSDB
  files on each central node.
* **dqlite database** (cluster state, CA certificate and key, join tokens,
  configuration), replicated to all members via RAFT.
* **TLS private keys and certificates**, on disk for each local service, and
  the CA key/cert in dqlite.

All of this data is accessible only to ``root``. No data is transmitted outside
the cluster unless explicitly configured (e.g. connecting to an external OVN
central cluster via :doc:`/reference/config/ovn-central-ips`). There is no
built-in log redaction, OVN/OVS logs may contain IP addresses and port numbers
but not user payload.

Further reading
---------------

* :doc:`/how-to/tls`: managing certificates and upgrading from plaintext
* :doc:`/reference/cryptography`: algorithms, key sizes, certificate lifecycle
* :doc:`/reference/security`: vulnerability reporting and response process
* :doc:`/reference/hardening`: hardening recommendations
