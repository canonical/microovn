============
Cryptography
============

Transport layer security (TLS)
------------------------------
All network endpoints exposed by MicroOVN services are secured using multiple
components of the `TLS protocol`_, including encryption, authentication and
integrity.  Through the use of the `Ubuntu OpenSSL`_ packages, TLS versions
below 1.2 are disabled for security reasons.

There are two self-signed certificate authorities in use, one for the
`MicroCluster`_ based ``microovnd`` daemon, another for the `OVN`_ daemons.
These are initialised during the initial bootstrap of the cluster.

Keys are generated using a 384 bit `Elliptic Curve`_ algorithm often referred
to as P-384.

MicroOVN's ``Go`` code uses package `crypto`_  from standard library to parse,
generate and validate TLS certificates and associated cryptographic keys.

Both sets of daemons are by default configured to make use of TLS to encrypt
on the wire communication, as well as using certificate data for authenticating
and verifying remote peers, ensuring only trusted components can participate
in the cluster.

User interaction
----------------

MicroOVN exposes limited actions for user to interact with TLS certificates
used by the OVN services. Note that no mechanism is provided to interact with
certificates used internally by MicroOVN API endpoints. For more information
about how to manage OVN certificates, please see :doc:`Working with TLS
</how-to/tls>`, specifically sections:

  * :ref:`Re-issue certificates <issue_certificates>`
  * :ref:`Manage Certificate Authority <manage_ca>`

.. _certificates_lifecycle:

OVN Certificate lifecycle
-------------------------

OVN service certificates that are automatically provisioned by MicroOVN have
the following lifespans:

* CA certificate: 10 years
* OVN service/client certificate: 2 years

MicroOVN runs daily checks for certificate lifespan validity. When a
certificate is within 10 days of expiration, it will be automatically renewed.

.. note::
   CA certificate is automatically renewed only if it's automatically generated
   by the MicroOVN. User-supplied CA certificate is not automatically renewed
   and needs to be manually updated by the user via
   :command:`certificates set-ca`

Data at rest
------------

While MicroOVN ensures that data is transmitted securely over the network between
its various endpoints, data on disk is stored unencrypted under the
``/var/snap/microovn/common/data`` directory. Access to this directory is
restricted to the `root` user only. Potentially sensitive data in there
includes:

  * OVN and OVS databases
  * OVN certificates and private keys

.. LINKS
.. _crypto: https://pkg.go.dev/crypto
.. _Elliptic Curve: https://en.wikipedia.org/wiki/Elliptic-curve_cryptography
.. _MicroCluster: https://github.com/canonical/microcluster
.. _OVN: https://docs.ovn.org/en/latest/
.. _TLS protocol: https://datatracker.ietf.org/doc/html/rfc8446
.. _Ubuntu OpenSSL: https://ubuntu.com/server/docs/openssl
