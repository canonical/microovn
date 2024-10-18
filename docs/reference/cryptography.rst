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

Both sets of daemons are by default configured to make use of TLS to encrypt
on the wire communication, as well as using certificate data for authenticating
and verifying remote peers, ensuring only trusted components can participate
in the cluster.

.. LINKS
.. _Elliptic Curve: https://en.wikipedia.org/wiki/Elliptic-curve_cryptography
.. _MicroCluster: https://github.com/canonical/microcluster
.. _OVN: https://docs.ovn.org/en/latest/
.. _TLS protocol: https://datatracker.ietf.org/doc/html/rfc8446
.. _Ubuntu OpenSSL: https://ubuntu.com/server/docs/openssl
