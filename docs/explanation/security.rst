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

Security event logging
----------------------

MicroOVN emits structured security event log entries in accordance with the
`OWASP Application Logging Vocabulary`_ and `Logging Cheat Sheet`_. Every
security event carries the following structured fields:

``security``
  Always ``true`` — allows filtering security events from operational logs.
``category``
  One of ``AUTHN``, ``AUTHZ`` or ``SYS``.
``event``
  An OWASP vocabulary identifier (e.g. ``authn_password_changed``,
  ``authz_admin``, ``sys_startup``).

Additional context fields (``node``, ``service``, ``action``, ``subject``)
are included where applicable. All entries are emitted through the same
framework used by the rest of MicroOVN, so they appear in the daemon's
standard log output (``journalctl`` for a snap installation).

Covered events
~~~~~~~~~~~~~~

Authentication [AUTHN]
^^^^^^^^^^^^^^^^^^^^^^

Because MicroOVN uses mutual TLS (mTLS) exclusively, the OWASP authentication
vocabulary is mapped to TLS certificate lifecycle operations:

.. list-table::
   :header-rows: 1
   :widths: 30 30 40

   * - OWASP event
     - MicroOVN mapping
     - Example log
   * - ``authn_password_changed``
     - CA or service certificate (re)issued
     - ``category=AUTHN event=authn_password_changed subject=CA auto_renew=true msg="CA certificate generated and stored"``

Authorization [AUTHZ]
^^^^^^^^^^^^^^^^^^^^^

MicroOVN has no per-user RBAC, all mutating operations are therefore logged as administrative activity:

.. list-table::
   :header-rows: 1
   :widths: 30 30 40

   * - OWASP event
     - MicroOVN mapping
     - Example log
   * - ``authz_admin``
     - Cluster join / leave
     - ``category=AUTHZ event=authz_admin action=cluster_join node=node-2 msg="Node 'node-2' joined cluster"``
   * - ``authz_admin``
     - Service enable / disable
     - ``category=AUTHZ event=authz_admin action=enable_service service=central node=node-1 msg="Enabling service 'central' on node 'node-1'"``
   * - ``authz_admin``
     - Configuration change (set / delete)
     - ``category=AUTHZ event=authz_admin action=config_set key=ovn.central-ips msg="Setting configuration key 'ovn.central-ips'"``
   * - ``authz_admin``
     - CA regeneration or custom CA upload
     - ``category=AUTHZ event=authz_admin action=regenerate_ca msg="CA certificate regeneration requested via API"``

System [SYS]
^^^^^^^^^^^^

.. list-table::
   :header-rows: 1
   :widths: 30 30 40

   * - OWASP event
     - MicroOVN mapping
     - Example log
   * - ``sys_startup``
     - Daemon start (``OnStart`` hook, fires on every start including post-bootstrap)
     - ``category=SYS event=sys_startup node=node-1 msg="MicroOVN daemon starting on 'node-1'"``
   * - ``sys_shutdown``
     - Node leaving cluster
     - ``category=SYS event=sys_shutdown node=node-2 msg="Node 'node-2' shutting down OVN services before departure"``

Events not applicable
~~~~~~~~~~~~~~~~~~~~~

The following OWASP events do not apply to MicroOVN due to its architecture
and are intentionally not implemented:

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Event category
     - Reason
   * - Successful / Failed Login
     - MicroOVN uses mTLS only. The TLS handshake is handled inside the
       ``microcluster`` library; successful and failed connection attempts
       are logged by the TLS layer, not by MicroOVN application code.
   * - Account Lockout
     - There are no user accounts. Identity is certificate-based and there
       is no lockout mechanism.
   * - Token Created / Deleted / Revoked / Reused
     - Join token lifecycle is handled entirely within the ``microcluster``
       library. ``microovn cluster add`` calls ``microcluster``
       ``NewJoinToken`` API. Tokens are single-use and expire automatically.
       MicroOVN application code has no visibility into any of these
       transitions.
   * - Unauthorized Access Attempt
     - Requests from untrusted clients are rejected by the ``microcluster``
       TLS listener before reaching MicroOVN handlers (all endpoints set
       ``AllowUntrusted: false``). The rejection is logged at the framework
       level.
   * - User Created / Updated
     - There are no user accounts. Node membership (join / leave) is the
       closest equivalent and is covered under ``authz_admin``.
   * - System Restart / Crash
     - Snap service restarts are managed by ``systemd``; crash recovery is
       handled by the snap runtime. MicroOVN does not have in-process
       restart or crash-handler hooks.
   * - System Monitoring Disabled
     - MicroOVN does not provide a monitoring subsystem that can be
       selectively disabled.

.. _OWASP Application Logging Vocabulary: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Vocabulary_Cheat_Sheet.html
.. _Logging Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html

Further reading
---------------

* :doc:`/how-to/tls`: managing certificates and upgrading from plaintext
* :doc:`/reference/cryptography`: algorithms, key sizes, certificate lifecycle
* :doc:`/reference/security`: vulnerability reporting and response process
* :doc:`/reference/hardening`: hardening recommendations
