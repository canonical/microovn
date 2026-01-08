========================
OVN integration with BGP
========================

OVN fully enabled support for integration with BGP in version ``25.03``. The
support includes the ability to redirect BGP control-plane traffic to a
specific ``Logical Switch Port`` and the ability of the ``ovn-controller`` to
advertise and learn routes via a `Linux VRF`_.

BGP control plane
-----------------

To avoid implementing the BGP protocol inside the OVN, the upstream project
decided to provide a way to redirect the BGP control plane traffic from a
``Logical Router Port`` to a ``Logical Switch Port``. As a result, an
off-the-shelf BGP daemon can be run inside a VRF, bound to the ``Logical Switch
Port``. Routing information is confined inside the VRF, leaving the routing
table of the main host unaffected. And finally, the BGP daemon appears to the
rest of the network as if it was bound to the ``Logical Router Port``.

The last point is important for the purpose of hardware offload and it also
allows us to use "BGP unnumbered" with BGP authentication. If the BGP
daemon acts as if it was bound to the ``Logical Router Port``, it
advertises its routes with the next hop address of the ``Logical Router Port``.
The data-plane traffic can be then accelerated via the hardware offload without
any further intervention.

Route advertisement and learning via VRF
----------------------------------------

The ``ovn-controller`` is capable of maintaining VRFs and using them to learn
and advertise routes.

When the chassis binds a ``Logical Router Port`` configured to advertise or
learn routes, it creates a VRF and inserts the routes to the VRF table. A
BGP daemon can be configured to use the OVN's VRF table, and announce the
routes to its peers. Conversely, when the BGP daemon learns routes from its
peers, it inserts them to the VRF, from which they are picked up by the
``ovn-controller`` and learned by the OVN.

What MicroOVN sets up
---------------------

MicroOVN can simplify the BGP integration setup described in the previous
sections. For more information on how to do it, see:
:doc:`How-To: Configure OVN BGP integration </how-to/bgp>`

To fully set up BGP redirection, MicroOVN requires following:

* one or more physical interfaces that provide connectivity to the external
  networks
* VRF table ID that the OVN will create and to which the internal routes will
  be redistributed.
* (Optional) AS number that will be used by the BGP daemon to identify itself.
  Note that if this is not provided, the BGP daemon won't be configured. You
  can choose to omit the ASN if you wish to configure the BGP daemon manually.

With this information provided, MicroOVN will then set up an OVN ``Logical
Router`` that will act as a gateway to the external networks. This is going to
be a "gateway router" with name ``lr-<hostname>-microovn``.

Each of the provided physical interfaces will be plugged to its own OVS bridge
and connected to a unique ``Logical Switch``. The bridge will be called
``br-<interface_name>`` and the switch will be
``ls-<hostname>-<interface_name>``. The ``Logical Router`` will then be
connected to the ``Logical Switches`` via ports named
``lrp-<hostname>-<interface_name>``. The router ports are not configured with
any IP address and rely on IPv6 link local address to talk to the hosts on the
external network.

After the external connectivity is set up, MicroOVN will create ``Logical
Switch Port`` named ``lsp-<hostname>-<interface_name>-bgp`` in each ``Logical
Switch``. This is the logical port to which the BGP traffic will be redirected.
On the system interface level, the redirected traffic is handled by a veth
pair. Ends of this pair are named ``v<interface_name>-brg`` and
``v<interface-name>-bgp``. The ``-brg`` end is plugged into the OVS integration
bridge and bound to the above mentioned ``Logical Switch Port``. The ``-bgp``
end is plugged to the VRF, where the BGP daemon can be bound to it.

MicroOVN will then configure required OVN options.

On the ``Logical Router`` that provides the external connectivity:

* ``dynamic-routing-vrf-id`` set to the value of the VRF table.
* ``dynamic-routing`` set to ``true``

On each ``Logical Router Port`` plugged to the external network:

* ``dynamic-routing-maintain-vrf`` set to ``true``
* ``dynamic-routing-redistribute`` set to ``nat,lb``
* ``dynamic-routing-port-name`` set to a unique identifier that is used
  as a key in the ``dynamic-routing-port-mapping`` in the OVS database
* ``routing-protocols`` set to ``BGP,BFD``
* ``routing-protocol-redirect`` set to the name of the ``Logical Switch Port`` to
  which the traffic will be redirected

In the local Open vSwitch database:

* External ID ``dynamic-routing-port-mapping`` in the ``Open_vSwitch`` table
  that maps ``dynamic-routing-port-name`` option on ``Logical Router Ports`` to
  system interfaces on which the routes are learned

Bundled BGP daemon
------------------

MicroOVN comes bundled with `BIRD Routing Daemon`_ that implements BGP and BFD
protocols. Once the BGP integration is enabled, BIRD can be used to listen on the newly
created interfaces in the VRF and form connections with neighbours on the
external networks.

Automatic daemon configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

MicroOVN is capable of automatically configuring BIRD's BGP and BFD services.
If user provides ``asn`` config option when enabling BGP in MicroOVN, it will
configure BIRD to listen on each "BGP redirect" system interface. This is a
very opinionated configuration that uses "BGP unnumbered" mode for automatic
neighbour discovery. In effect it looks something like this:

.. code-block:: none

   protocol bgp microovn_eth1 {
       router id 192.0.2.10;
        interface "veth1-bgp";
        vrf "ovnvrf10";
        local as 4210000000;
        neighbor range fe80::/10 external;
        dynamic name "dyn_microovn_eth1_";
        ipv4 {
		    next hop self ebgp;
		    extended next hop on;
		    require extended next hop on;
		    import all;
		    export filter no_default_v4;
	    };
	    ipv6 {
		    import all;
		    export filter no_default_v6;
	    };
	    bfd {
		    # We only want to use BFD for liveness and failure detection if
		    # our peer has it configured.
		    passive yes;
	    };
   }

.. note::

   There's currently a quirk in BIRD's behaviour. When it's configured in the
   dynamic mode (by using ``neighbor range ...``), it doesn't try to discover
   any neighbours on the link.

   This means that if you use BIRD in dynamic mode on both ends (in the
   MicroOVN and on the external network), they will never connect. The
   solution is to either configure neighbor explicitly on either end, or use
   other routing daemons that do perform active discovery, like `FRR`_.

Example topology
----------------

Below is a diagram of an example topology. It's a single MicroOVN node "movn1",
connected to two external networks via physical interfaces "eth1" and "eth2".

.. image:: /static/bgp/bgp-multilink-light.svg
   :class: only-light

.. image:: /static/bgp/bgp-multilink-dark.svg
   :class: only-dark

.. LINKS
.. _BIRD Routing Daemon: https://bird.network.cz
.. _Linux VRF: https://docs.kernel.org/networking/vrf.html
.. _FRR: https://frrouting.org