=============================
Configure OVN BGP integration
=============================

Configuration of the OVN integration with BGP is a single-command process in
the MicroOVN, for more information about what's happening under the hood, see:
:doc:`Explanation: OVN integration with BGP </explanation/bgp-redirect>`.

Enable BGP integration
----------------------

In this example, we have a host connected to two external networks via
interfaces ``eth1`` and ``eth2``.

The only required configuration is specifying the external connection interfaces.
Both `VRF`_ table number and AS number are optional:

- If VRF is not specified, MicroOVN will automatically select an available VRF table ID
- If AS number is not specified, MicroOVN will automatically generate an unique ASN

.. important::

   Never use interface that provides actual host connectivity for the purpose
   of OVN BGP integration. These interfaces are meant for the OVN's traffic,
   they will be assigned to a OVS bridge and you will lose your connection to the host.

Basic usage with automatic configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To enable BGP integration with automatic VRF and ASN selection:

.. code-block:: none

   microovn enable bgp --config ext_connection=eth1,eth2

MicroOVN will automatically:

- Select an available VRF table ID
- Generate a unique ASN from the default range (4210000000-4294967294)

.. note::

   The default ASN range reserves the first IDs of the full `RFC 6996`_ private range (4200000000-4209999999)
   for lower tier network infrastructure components (switches, routers, top-of-rack equipment).

Advanced configuration
~~~~~~~~~~~~~~~~~~~~~~

You can explicitly specify VRF table ID, ASN, or provide a custom ASN range, in any combination.

Specify both VRF and ASN explicitly:

.. code-block:: none

   microovn enable bgp --config ext_connection=eth1,eth2 --config vrf=10 --config asn=4210000000

Provide a custom ASN range for automatic selection:

.. code-block:: none

   microovn enable bgp --config ext_connection=eth1,eth2 --config asn_range=4210000000-4210999999

.. important::

   When using a custom ASN range, provide one sufficiently large to avoid collisions between cluster members.
   Take into account that the ASN that would be allocated for a certain cluster member will never be reused by
   another, even if the first one is removed from the cluster.

You will receive positive confirmation message in the CLI and the setup is
done.

Inspect the changes
~~~~~~~~~~~~~~~~~~~

If you are interested in changes that the MicroOVN made to the system, We can
inspect the new interfaces that were created.

.. code-block:: none

   ip link

The output will show that we have a new VRF device and two veth pairs for
the BGP control-plane traffic (one pair for each external interface):

.. code-block:: none

   <snipped preceding output>
   13: ovnvrf10: <NOARP,MASTER,UP,LOWER_UP> mtu 65575 qdisc noqueue state UP mode DEFAULT group default qlen 1000
       link/ether 2e:25:b7:f0:f7:21 brd ff:ff:ff:ff:ff:ff
   14: veth1-brg@veth1-bgp: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master ovs-system state UP mode DEFAULT group default qlen 1000
       link/ether 46:d7:9e:a5:81:fa brd ff:ff:ff:ff:ff:ff
   15: veth1-bgp@veth1-brg: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master ovnvrf10 state UP mode DEFAULT group default qlen 1000
       link/ether 02:0e:da:a6:c4:28 brd ff:ff:ff:ff:ff:ff
   16: veth2-brg@veth2-bgp: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master ovs-system state UP mode DEFAULT group default qlen 1000
       link/ether ba:1e:35:32:8b:36 brd ff:ff:ff:ff:ff:ff
   17: veth2-bgp@veth2-brg: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master ovnvrf10 state UP mode DEFAULT group default qlen 1000
       link/ether 02:66:5b:b1:78:6e brd ff:ff:ff:ff:ff:ff
   <snipped remaining output>

And since we requested auto-configuration of BGP daemon, we can check the
BIRD configuration found at ``/var/snap/microovn/common/data/bird/bird.conf``.
There should be two "bgp" instances instances.

.. code-block:: none

   <snipped preceding output>
   protocol bgp microovn_eth2 {
       router id 192.0.2.10;
	    interface "veth2-bgp";
	    vrf "ovnvrf10";
	    local as 4210000000;
	    neighbor range fe80::/10 external;
	    dynamic name "dyn_microovn_eth2_";
   <snipped remaining output>

and

.. code-block:: none

   <snipped preceding output>
   protocol bgp microovn_eth1 {
       router id 192.0.2.10;
	    interface "veth1-bgp";
	    vrf "ovnvrf10";
	    local as 4210000000;
	    neighbor range fe80::/10 external;
	    dynamic name "dyn_microovn_eth1_";
   <snipped remaining output>

.. note::

   Note that for then neighbour configuration, we are not using the names of
   actual physical interfaces (e.g. ``eth1``), but the names of the interfaces
   that were created for BGP redirect (e.g. ``eth1-bgp``)

If there are BGP neighbours already running and configured on the external
networks, you can validate that they successfully established connections:

.. code-block:: none

   microovn.birdc show protocols

The output should contain established BGP sessions.

.. code-block:: none

   <snipped preceding output>
   microovn_eth1 BGP        ---        start  15:21:14.086  Passive
   microovn_eth2 BGP        ---        start  15:21:14.086  Passive
   dyn_microovn_eth1_1 BGP        ---        up     15:37:34.578  Established
   dyn_microovn_eth2_1 BGP        ---        up     15:38:00.689  Established
   <snipped remaining output>

.. _manual_bgp:

Manual BGP daemon configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In case that the automatic BIRD configuration provided by MicroOVN does not
suit your needs, you can just use the ``--manual-bgpd-config`` flag when
enabling BGP, for example:

.. code-block:: none

   microovn enable bgp --config ext_connection=eth1,eth2 --manual-bgpd-config

With this flag, MicroOVN won't configure the built-in
BIRD daemon, Allowing you to perform manual configuration or use entirely
different BGP daemon.

Disable BGP integration
-----------------------

To disable BGP integration, simply run:

.. code-block:: none

   microovn disable bgp

This will remove all VRF tables, virtual interfaces, OVS bridges, Logical
Switches and Logical Routers that were created when the integration was
enabled.

MicroOVN will also backup and reset BIRD startup configuration. The current
configuration file will be backed up in the same directory under name
``bird.conf_<unix_timestamp>`` and then replaced with the default
BIRD configuration.

.. LINKS
.. _VRF: https://docs.kernel.org/networking/vrf.html
.. _RFC 6996: https://datatracker.ietf.org/doc/html/rfc6996
