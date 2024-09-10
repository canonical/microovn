=============================
Configure OVN BGP integration
=============================

.. important::

   **Experimental feature**: OVN BGP integration is currently an experimental
   feature in both MicroOVN and upstream OVN. As such it is still undergoing
   testing and is subject to changes or removal in the future.

Configuration of the OVN integration with BGP is a single-command process in
the MicroOVN, for more information about what's happening under the hood, see:
:doc:`Explanation: OVN integration with BGP </explanation/bgp-redirect>`.

Enable BGP integration
----------------------

In this example, we have a host connected to two external networks:

* ``10.0.10.0/24`` via interface ``eth1``
* ``10.0.20.0/24`` via interface ``eth2``

We are going to need one free IPv4 address on each network to give to the
OVN ``Logical Router``, in this example we will use the first available
address. We will also need an unused VRF table number, in this example we will
use ``10``. Final, and optional, thing we will use is an AS number, we will
simply pick ``1``, and by providing it, we will let MicroOVN know that we want
it to auto-configure BGP daemons with this ``ASN``.

.. important::

   Never use interface that provides actual host connectivity for the purpose
   of OVN BGP integration. These interfaces will be assigned to the OVS bridge
   and you will lose your connection to the host.

To enable BGP integration run:

.. code-block:: none

   microovn enable bgp --config ext_connection=eth1:10.0.10.1/24,eth2:10.0.20.1/24 --config vrf=10 --config asn=1

You will receive positive confirmation message in the CLI and the setup is
done. We can also inspect the new interfaces that were created.

.. code-block:: none

   ip link

The output will show that we have a new VRF device and BGP redirect ports for
each network we specified:

.. code-block:: none

   <snipped preceding output>

   20: ovnvrf10: <NOARP,MASTER,UP,LOWER_UP> mtu 65575 qdisc noqueue state UP mode DEFAULT group default qlen 1000
       link/ether c2:c3:11:b9:30:5c brd ff:ff:ff:ff:ff:ff
   21: eth1-bgp: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master ovnvrf10 state UNKNOWN mode DEFAULT group default qlen 1000
       link/ether 02:48:5a:cf:f6:47 brd ff:ff:ff:ff:ff:ff
   22: eth2-bgp: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master ovnvrf10 state UNKNOWN mode DEFAULT group default qlen 1000
       link/ether 02:ee:23:d2:88:14 brd ff:ff:ff:ff:ff:ff

   <snipped remaining output>

And since we requested auto-configuration of BGP daemons, we can check the FRR configuration:

.. code-block:: none

   microovn.vtysh -c "sh run"

Part of the output should show our new BGP instances:

.. code-block:: none

   !
   router bgp 1 vrf ovnvrf10
    neighbor eth1-bgp interface remote-as internal
    neighbor eth2-bgp interface remote-as internal
   !

.. note::

   Note that for then neighbour configuration, we are not using the names of
   actual physical interfaces (e.g. ``eth1``), but the names of the interfaces
   that were created for BGP redirect (e.g. ``eth1-bgp``)

If there are BGP neighbours already running and configured on the external
networks, you can validate that they successfully established connections:

.. code-block:: none

   microovn.vtysh -c "sh bgp vrf ovnvrf10 neighbors"

There will be a lot of output, but for both BGP neighbours, there should be
line indicating that the connection is in "Established" state. Snippet example:

.. code-block:: none

   BGP neighbor on eth1-bgp: fe80::216:3eff:fec8:8649, remote AS 1, local AS 1, internal link
     Local Role: undefined
     Remote Role: undefined
   Hostname: bgp-peer
     BGP version 4, remote router ID 10.75.224.199, local router ID 10.0.20.1
     BGP state = Established, up for 00:38:14
     Last read 00:00:14, Last write 00:00:14

Manual BGP daemon configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In case that the automatic FRR configuration provided by MicroOVN does not
suite your needs, you can just omit the ``--config asn=<ASN>`` option when
enabling BGP. Without that option, MicroOVN won't attempt to do any
configuration changes to the FRR and you can proceed with your own manual
configuration.

Disable BGP integration
-----------------------

To disable BGP integration, simply run:

.. code-block:: none

   microovn disable bgp

This will remove all VRF tables, virtual interfaces, OVS bridges, Logical
Switches and Logical Routers that were created when the integration was
enabled.

MicroOVN will also backup and reset FRR startup configuration. The current
"startup" configuration file will be backed up in the same directory under name
``frr.conf_<unix_timestamp>`` and then replaced with the default, empty,
FRR configuration.
