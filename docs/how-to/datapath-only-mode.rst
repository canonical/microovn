==================
Datapath-only Mode
==================

This is a mode of operation in which MicroOVN cluster runs only ``switch`` and
``chassis`` service, with ``central`` service being provided externally by another
MicroOVN cluster or by some other OVN deployment.

.. important::

   This is not an equivalent of the OVN Interconnect feature. It does not allow
   connecting two full OVN deployments. Instead it is meant to help with migrations,
   or in hybrid deployments, where OVN central services are deployed by some other
   method.

Cluster setup
-------------

We will start by deploying the MicroOVN cluster without the ``central`` service. Our
hosts that run MicroOVN are called ``node-1``, ``node-2`` and ``node-3``. Setup of
the external OVN central cluster is not covered by this how-to. For the sake of this
guide we assume that there are OVN Southbound and OVN Northbound services running on
their default ports on IPs:

* 10.0.0.1
* 10.0.0.2
* 10.0.0.3

.. note::

   If you already have MicroOVN deployed, see
   :doc:`Service Control </how-to/service-control>` about how to disable ``central``
   service on the running nodes, and :doc:`MicroOVN services </reference/services>`
   regarding implications of removing all ``central`` services on the running cluster.

Obtain CA certificate and private key for the MicroOVN
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Because the MicroOVN enforces encrypted communication between OVN services, we need
to ensure that OVN services running on the MicroOVN cluster and on the external OVN
central cluster are mutually trusted.

Obtain intermediate CA certificate and private key from the CA that issued certificates
for the external OVN central cluster and place it in the ``/var/snap/microovn/common/``
on the ``node-1``, under names ``ca.cert`` and ``ca.key``.

Bootstrap MicroOVN cluster
~~~~~~~~~~~~~~~~~~~~~~~~~~

First we bootstrap the MicroOVN cluster on the ``node-1`` and generate tokens, so that
``node-2`` and ``node-3`` can join.

.. code-block::

   sudo microovn init

We will be taken through an initialisation process that will look something like this:

.. code-block::

   Please choose the address MicroOVN will be listening on [default=10.75.224.213]:
   Would you like to create a new MicroOVN cluster? (yes/no) [default=no]: yes
   Please select comma-separated list services you would like to enable on this node (central/chassis/switch) or let MicroOVN automatically decide (auto) [default=auto]: switch,chassis
   Please choose a name for this system [default=node-1]:
   Would you like to define a custom encapsulation IP address for this member? (yes/no) [default=no]:
   Would you like to provide your own CA certificate and private key for issuing OVN TLS certificates? (yes/no) [default=no]: yes
   Please enter the path to the CA certificate file: /var/snap/microovn/common/ca.cert
   Please enter the path to the CA private key file: /var/snap/microovn/common/ca.key
   Would you like to add additional servers to the cluster? (yes/no) [default=no]: yes
   What's the name of the new MicroOVN server? (empty to exit): node-2
   <NODE_2_TOKEN>
   What's the name of the new MicroOVN server? (empty to exit): node-3
   <NODE_3_TOKEN>

Then we join the cluster with ``node-2`` and ``node-3``

.. code-block::

   sudo microovn init

Similar dialogue takes us through the joining process, the most notable difference is that
we will select ``no`` when asked about whether we want to create a new cluster.

.. code-block::

   Please choose the address MicroOVN will be listening on [default=10.75.224.175]:
   Would you like to create a new MicroOVN cluster? (yes/no) [default=no]: no
   Please select comma-separated list services you would like to enable on this node (central/chassis/switch) or let MicroOVN automatically decide (auto) [default=auto]: chassis,switch
   Please enter your join token: <NODE_TOKEN>
   Would you like to define a custom encapsulation IP address for this member? (yes/no) [default=no]:

External central configuration
------------------------------

Now that the MicroOVN cluster is deployed without the central services, we can configure
it to connect to the external OVN central cluster. On any of the nodes, run following
command:

.. code-block::

   sudo microovn config set ovn.central-ips "10.0.0.1,10.0.0.2,10.0.0.3"

Verification
------------

To verify that the configuration was successfully applied, we can check that our
chassis successfully registered with the OVN Southbound database.

.. code-block::

   sudo microovn.ovn-sbctl show

The output should look something like this:

.. code-block::

   Chassis node-1
       hostname: node-1.lxd
       Encap geneve
           ip: "10.75.224.213"
           options: {csum="true"}
   Chassis node-3
       hostname: node-3.lxd
       Encap geneve
           ip: "10.75.224.138"
           options: {csum="true"}
   Chassis node-2
       hostname: node-2.lxd
       Encap geneve
           ip: "10.75.224.175"
           options: {csum="true"}

This proves that our client commands (``ovn-sbctl``) are able to connect to the external
OVN central and that ``ovn-controller`` services running on nodes in the MicroOVN cluster
got registered in the external Southbound database.
