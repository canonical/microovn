================
Working with TLS
================

Starting with snap revision ``111``, new deployments of MicroOVN use TLS
encryption by default. A self-signed CA certificate is used to issue
certificates to all OVN services that require it. They provide authentication
and encryption for OVSDB communication. The CA certificate is generated during
cluster initialisation (:command:`cluster bootstrap` command).

In the current implementation, self-provisioned certificates are the only mode
available. Future releases may include support for externally provided
certificates.

.. warning::

   The certificate and private key generated for the self-provisioned CA are
   currently stored unencrypted in the database on every cluster member. If an
   attacker gains access to any cluster member, they can use the CA to issue
   valid certificates that will be accepted by other cluster members.

Certificates CLI
----------------

MicroOVN exposes a few commands for basic interaction with TLS certificates.

List certificates
~~~~~~~~~~~~~~~~~

To list currently used certificates:

.. code-block:: none

   microovn certificates list

Example output:

.. code-block:: none

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

This command does not perform any certificate validation, it only ensures that
if a service is available on the node, the file that should contain a
certificate is in place.

Re-issue certificates
~~~~~~~~~~~~~~~~~~~~~

The :command:`certificates reissue` command is used to interact with OVN
services on the local host; it does not affect peer cluster members.

.. important::

   Services must be running in order to be affected by the
   :command:`certificates reissue` command. For example, running
   :command:`certificates reissue ovnnb` on a member that does not run this
   service is expected to fail.

To re-issue a certificate for a single service:

.. code-block:: none

   microovn certificates reissue <ovn_service_name>

To re-issue certificates for all services, the ``all`` argument is supported:

.. code-block:: none

   microovn certificates reissue all

Valid service names can be discovered with the ``--help`` option:

.. code-block:: none

   microovn certificates reissue --help

Regenerate PKI for the cluster
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The :command:`certificates regenerate-ca` command is used to issue a new CA
certificate and new certificates for every OVN service in the cluster:

.. code-block:: none

   microovn certificates regenerate-ca

This command replaces the current CA certificate and notifies all cluster
members to re-issue certificates for all their services. The command's output
will include evidence of successfully issued certificates for each cluster
member.

.. warning::

   A new certificate must be issued successfully for every service on every
   member. Any failure will result in subsequent communication errors for that
   service within the cluster.

Certificate lifecycle
---------------------

Certificates that are automatically provisioned by MicroOVN have the following
lifespans:

* CA certificate: 10 years
* OVN service certificate: 2 years

MicroOVN runs daily checks for certificate lifespan validity. When a
certificate is within 10 days of expiration, it will be automatically renewed.

Upgrade from plaintext to TLS
-----------------------------

Plaintext communication is used when MicroOVN is initially deployed with a snap
revision of less than ``111``, and there's no way to automatically convert to
encrypted communication. The following manual steps are needed to upgrade from
plaintext to TLS:

* ensure that all MicroOVN snaps in the cluster are upgraded to, at least,
  revision ``111``
* run ``microovn certificates regenerate-ca`` on one of the cluster members
* run ``sudo snap restart microovn.daemon`` on **all** cluster members

Once this is done, OVN services throughout the cluster will start listening on
TLS-secured ports.

Common issues
-------------

This section contains some well known or expected issues that you can encounter.

I'm getting ``failed to load certificates`` error
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

If you run commands like :command:`microovn.ovn-sbctl` and you get complaints
about missing certificates while the rest of the commands seem to work fine.

Example:

.. code-block:: none

   microovn.ovn-sbctl show

Example output:

.. code-block:: none

   2023-06-14T15:09:31Z|00001|stream_ssl|ERR|SSL_use_certificate_file: error:80000002:system library::No such file or directory
   2023-06-14T15:09:31Z|00002|stream_ssl|ERR|SSL_use_PrivateKey_file: error:10080002:BIO routines::system lib
   2023-06-14T15:09:31Z|00003|stream_ssl|ERR|failed to load client certificates from /var/snap/microovn/common/data/pki/cacert.pem: error:0A080002:SSL routines::system lib
   Chassis microovn-0
       hostname: microovn-0
       Encap geneve
           ip: "10.5.3.129"
           options: {csum="true"}

This likely means that your MicroOVN snap got upgraded to a version that
supports TLS, but it requires some manual upgrade steps. See section `Upgrade
from plaintext to TLS`_.
