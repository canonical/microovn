===================
``ovn.central-ips``
===================

.. list-table::
   :header-rows: 0

   * - Key
     - ovn.central-ips
   * - Type
     - String
   * - Scope
     - Cluster
   * - Description
     - Comma-separated list of IP addresses of the OVN central cluster
   * - Example
     - 10.0.0.1,10.0.0.2,10.0.0.3

By default, MicroOVN assumes that it manages the whole OVN stack from the
OVN Northbound database down to the Open vSwitch. This option can be used to change
that. It instructs MicroOVN to connect to an external OVN central cluster (OVN Northbound
and OVN Southbound databases). Setting this option causes:

* ``ovn-controller`` on all nodes with ``chassis`` service enabled to connect and register
  to the OVN Southbound database specified by this option
* ``ovn-nbctl`` and ``ovn-sbctl`` client commands to connect to their respective database
  specified by this option.

This option is applied cluster-wide. Setting it on any of the nodes will apply necessary
changes across the whole MicroOVN cluster.

Prerequisites
-------------

There are few things that need to be taken into consideration when applying this
configuration option.

Certificate Authority match
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Since MicroOVN enforces encrypted communication between clients and services, the
external OVN central cluster needs to use SSL/TLS on its database and API endpoints.
The certificates used by the external cluster also need to be trusted by the clients
in the MicroOVN cluster (and vice versa). In practice this means obtaining an intermediate
CA certificate and key from the CA that issued certificates for the external cluster, and
setting it as a CA certificate/key in the MicroOVN.
See :doc:`Working with TLS </how-to/tls>` about how to set user-provided CA certificate and
key in the MicroOVN.

(Optional) Disable internal OVN central cluster
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

While it's not strictly necessary, if the MicroOVN deployment is using an external OVN
central cluster, it is usually unnecessary for any of the internal nodes to actually
run the ``central`` service. If any of your nodes still run the ``central`` service,
you can disable it. See :doc:`MicroOVN services </reference/services>` and
:doc:`Service Control </how-to/service-control>` about how to do it.
