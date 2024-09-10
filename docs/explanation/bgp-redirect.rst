========================
OVN integration with BGP
========================

OVN enabled support for integration with BGP in version ``24.09`` as an
experimental feature (`BGP redirect commit`_). It enables redirecting BGP
(and BFD) traffic from Logical Router Port to a Logical Switch Port. The
Logical Switch Port can be then bound to an OVS port that shows up as a real
network interface in the system. By doing so, it enables any routing daemons
to listen on this interface and effectively act as if the daemon was listening
on the Logical Router Port. Benefits of this type of integration are that it
enables hardware offloading, as well as running BGP in unnumbered mode.

MicroOVN can automate a lot of the steps to configure the BGP integration. For
more information on how to configure it, see:
:doc:`How-To: Configure OVN BGP integration </how-to/bgp>`

Route announcement and VRFs
---------------------------

The part that provides automatic leaking of OVN routes to the BGP daemon
has not been made part of the released OVN yet. However, as an experimental
feature, we are including an `OVN patch series`_ that is a candidate for this
feature in the upstream OVN.

With this patch applied, it's possible to instruct OVN to create a `Linux VRF`_
, an separate routing table to which it will leak its own routes. These routes
can be then picked up by BGP daemon and announced to its peers.

What MicroOVN sets up
---------------------

To fully set up BGP redirecting, MicroOVN requires following:

* one or more physical interfaces that provide connectivity to the external
  networks
* one free IPv4 address per external network that will be assigned to the
  OVN router for external connectivity.
* VRF table ID that the OVN will create and to which the internal routes will
  be leaked.

With these information provided, MicroOVN will then set up an OVN ``Logical
Router`` that will act as a gateway to the external networks. This is going to
be a "gateway router" with name ``lr-<hostname>-microovn``.

Each of the provided interfaces will be plugged to its own OVS bridge and
connected to a unique ``Logical Switch``. The bridge will be called
``br-<interface_name>`` and the switch will be
``ls-<hostname>-<interface_name>``. The ``Logical Router`` will then be
connected to the ``Logical Switches`` via ports named
``lrp-<hostname>-<interface_name>`` and assigned the provided free IPv4
address. This will effectively give external connectivity to the router
and vice versa.

After the external connectivity is set up, MicroOVN will create ``Logical
Switch Port`` named ``lsp-<hostname>-<interface_name>-bgp`` in each ``Logical
Switch``. This is the logical port to which the BGP traffic will be redirected.
It will be bound to an OVS port named ``<interface_name>-bgp`` that will show up
in the host system.

MicroOVN will then configure required OVN options:

* ``requested-tnl-key`` set to the value of the VRF table on the ``Logical
  Router``
* ``maintain-vrf`` set to ``true`` on each external ``Logical Router Port``
* ``redistribute-nat`` set to ``true`` on each external ``Logical Router
  Port``
* ``redistribute-lb-vips`` set to ``true`` on each external ``Logical Router
  Port``
* ``routing-protocols`` set to ``BGP,BFD`` on each external ``Logical Router
  Port``
* ``redistribute-lb-vips`` set to the name of the ``Logical Switch Port`` to
  which the traffic will be redirected on each external ``Logical Router
  Port``

As a last step, the redirect interfaces will be moved to the created VRF and the
feature is ready to be used.

Bundled BGP daemon
------------------

MicroOVN comes bundled with `FRR suite`_ that implements BGP and BFD daemons.
Once the BGP integration is enabled, it can be used to listen on the newly
created interfaces in the VRF and form connections with neighbours on the
external networks.

Automatic daemon configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

MicroOVN is capable of using FRR to automatically set up BGP daemons when then
integration is enabled. If user provides ``asn`` config option when enabling
BGP, MicroOVN will start daemons on each "BGP redirect" system interface it
created. This is a very opinionated configuration that uses "BGP unnumbered"
mode for automatic neighbour discovery. In effect it looks something like this:

.. code-block:: none

   !
   router bgp <ASN> vrf <vrf_name>
    # One neighbor declaration is added for each external connection
    neighbor <bgp-redirect-iface-1> interface remote-as internal
    neighbor <bgp-redirect-iface-2> interface remote-as internal
    ...
   exit
   !

FRR configuration persistence
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Whenever MicroOVN changes FRR config, it persists current "running"
configuration by copying it to "startup" configuration file.

When user disables BGP integration, the current "startup" configuration
file is backed up to ``frr.conf_<unix_timestamp>`` file and subsequently
replaced with the default empty configuration.

.. LINKS
.. _BGP redirect commit: https://github.com/ovn-org/ovn/commit/370527673c2b35c1b79d90a4e5052177e593a699
.. _OVN patch series: https://patchwork.ozlabs.org/project/ovn/patch/20240725140009.413791-1-fnordahl@ubuntu.com/
.. _Linux VRF: https://docs.kernel.org/networking/vrf.html
.. _FRR suite: https://frrouting.org