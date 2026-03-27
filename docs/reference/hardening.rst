=========
Hardening
=========

MicroOVN enforces TLS encryption and authentication on all network endpoints
and uses sane defaults wherever possible. This page documents areas where the
default posture can be strengthened and how MicroOVN relates to upstream OVN
security guidance.

Default protections
-------------------

The following are active out of the box:

* **Mutual TLS** on all OVN/OVS and MicroCluster network endpoints (since snap
  revision 111).
* **ECDSA P-384** keys with automatic daily renewal of expiring certificates.
* **Snap strict confinement** limiting filesystem and network access.
* **Root-only permissions** on all on-disk state
  (``/var/snap/microovn/common/data``).

Filesystem and disk
-------------------

Databases, private keys, and certificates are stored unencrypted on disk.
Consider enabling full-disk encryption on hosts that process sensitive
network policies.

Network exposure
----------------

MicroOVN listens on the following TCP ports. Restrict access to these ports
through firewalls or network segmentation to the set of hosts that need them:

.. list-table::
   :header-rows: 1

   * - Port
     - Service
     - Authentication
   * - 6641
     - OVN Northbound OVSDB
     - mTLS
   * - 6642
     - OVN Southbound OVSDB
     - mTLS
   * - 6643
     - OVN Northbound cluster (RAFT)
     - mTLS
   * - 6644
     - OVN Southbound cluster (RAFT)
     - mTLS
   * - 6081
     - Geneve tunnel (OVS datapath)
     - None (upstream OVN)
   * - 6686
     - MicroCluster REST API
     - mTLS
   * - 179
     - BGP (BIRD), when enabled
     - None by default

OVSDB access
------------

MicroOVN does not enable OVN's optional RBAC on the Northbound or Southbound
databases. Any client presenting a valid TLS certificate signed by the cluster
CA has full read/write access. In multi-tenant or shared environments, consider
network-level restrictions to limit which hosts can reach the OVSDB ports.

Upstream OVN security guidance
------------------------------

The OVN upstream documentation covers security topics such as RBAC for the
Southbound database, TLS configuration, and OVSDB access control (see the
`OVN RBAC tutorial`_ for details). MicroOVN's opinionated design satisfies or
deviates from this guidance as follows:

* **TLS**: upstream defaults to plaintext; MicroOVN enables TLS by default.
* **RBAC**: upstream supports optional Southbound RBAC; MicroOVN does not
  configure it.
* **Certificate management**: upstream leaves PKI to the operator; MicroOVN
  auto-provisions a self-signed CA and manages the full certificate lifecycle.

.. _OVN RBAC tutorial: https://docs.ovn.org/en/latest/tutorials/ovn-rbac.html

BGP integration
---------------

MicroOVN provides a way to integrate OVN natively with BGP routers on the
external networks. See :doc:`Configure OVN BGP integration </how-to/bgp>`
page for more information. When the integration is enabled with the ``--asn``
option specified, MicroOVN will auto-configure a `BIRD 3`_ BGP service to listen
on connections from the physical external network. This auto-configured BGP
daemon has a very lax security settings, most importantly it:

* doesn't perform peer authentication (see `RFC 2385`_)
* doesn't employ RPKI to validate route advertisements (see `RFC 6480`_)
* doesn't apply any route filtering on learned routes
* does connect to the first peer it finds on the external link

BGP security is a very broad topic that's out of scope for this document, but
the above points should cover basics when deploying BGP daemons in an
environment where the peers can't be necessarily trusted.

If the user desires any of the above security features, they are advised to
omit the ``--asn`` option when enabling the BGP integration. This will allow
them to bind any external BGP daemon to the interface inside the VRF created
by the MicroOVN. Then they will be able to tailor the daemon configuration
to their specific security needs.

.. LINKS
.. _BIRD 3: https://bird.network.cz/?get_doc&f=bird.html&v=30
.. _RFC 2385: https://datatracker.ietf.org/doc/html/rfc2385
.. _RFC 6480: https://datatracker.ietf.org/doc/html/rfc6480
