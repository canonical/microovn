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

Once this is done, OVN API services throughout the cluster will start listening
on TLS-secured ports. However, the process is not complete yet because OVN
Southbound and Northbound database clusters themselves are not capable of
automatically switching to TLS communication in existing clusters. Both
database clusters need to be manually switched over by individually removing
cluster members that use ``tcp`` connection and reconnecting them with ``ssl``.
This process technically replaces every member in the original cluster, but
because we are doing it gradually, cluster data remains intact.

Let's assume that we have a 3 node cluster. We'll start with switching over
the ``OVN Northbound`` cluster.

**Preparation**: We will be running commands on multiple nodes throughout this
process, it is recommended to open a separate shell on each node and keep it
open with following variables exported:

.. code-block:: none
    CONTROL_SOCKET=/var/snap/microovn/common/run/ovn/ovnnb_db.ctl
    DB=OVN_Northbound
    DB_FILE=/var/snap/microovn/common/data/central/db/ovnnb_db.db
    PORT=6643

**1st step**: Leave cluster with a single member:

.. code-block:: none

    # node-1
    microovn.ovn-appctl -t $CONTROL_SOCKET cluster/leave $DB

**2nd step**: Make sure that member properly left the cluster by inspecting
cluster status on nodes 2 and 3 and ensuring that node 1 is no longer part of
the cluster:

.. code-block:: none

    # run on node-2 and node-3
    microovn.ovn-appctl -t /var/snap/microovn/common/run/ovn/ovnnb_db.ctl cluster/status OVN_Northbound

**3rd step**: Clean up remaining DB files on node 1:

.. code-block:: none

    # node-1
    snap stop microovn.central
    rm $DB_FILE

**4th step**: Rejoin the cluster with node 1, using ``ssl`` as protocol for
local listening port. Notice that we will still use ``tcp`` as a protocol for
remote cluster connection because no other node listens on ``ssl`` yet. This
will get fixed automatically when other cluster members switch to ``ssl``:

.. code-block:: none

    # node-1
    microovn.ovsdb-tool join-cluster $DB_FILE $DB ssl:<local_ip>:$PORT tcp:<node_2_ip>:$PORT
    snap restart microovn.central

**5th step**: Monitor cluster as it converges to stable state. Use following
command to monitor cluster until it indicates three members and field
``Entries not yet applied`` reaches 0:

.. code-block:: none

    # node-1
    microovn.ovn-appctl -t $CONTROL_SOCKET cluster/status $DB

Now that node 1 successfully transitioned to TLS we can repeat the same steps
on node 2 and then on node 3. The only difference is in **4th step** where we
will use protocol ``ssl`` and IP of a node 1 as last arguments for
``microovn.ovsdb-tool`` command. To save you some searching and replacing,
here are the revised commands for the **4th step** to be used on node 2 and 3:

.. code-block:: none

    # node-{2,3}
    microovn.ovsdb-tool join-cluster $DB_FILE $DB ssl:<local_ip>:$PORT ssl:<node_1_ip>:$PORT
    snap restart microovn.central

After all three nodes transitioned to TLS usage, you can once again run:

.. code-block:: none

    # any node
    microovn.ovn-appctl -t $CONTROL_SOCKET cluster/status $DB

to verify that all three cluster members are using ``ssl`` as their connection
protocol.

This whole process needs to be repeated again for ``OVN Southbound`` cluster.
Steps and commands are the same, just with different set of variables configured
in the **Preparation** step:

.. code-block:: none

    CONTROL_SOCKET=/var/snap/microovn/common/run/ovn/ovnsb_db.ctl
    DB=OVN_Southbound
    DB_FILE=/var/snap/microovn/common/data/central/db/ovnsb_db.db
    PORT=6644

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
