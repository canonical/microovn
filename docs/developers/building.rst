======================================
Build and install MicroOVN from source
======================================

This how-to contains steps needed for building MicroOVN from its source code.
This is useful, for example, if you want to contribute to the MicroOVN and you
want to test your changes locally.

Build requirements
------------------

MicroOVN is distributed as a snap and the only requirements for building it
are ``Make`` and ``snapcraft``. You can install them with:

.. code-block:: none

   sudo apt install make
   sudo snap install snapcraft --classic

Snapcraft requires ``LXD`` to build snaps. So if your system does not have LXD
installed and initiated, you can check out either `LXD getting started
guides`_ or go with following default setup:

.. code-block:: none

   sudo snap install lxd
   lxd init --auto

Build MicroOVN
--------------

To build MicroOVN, go into the repository's root directory and run:

.. code-block:: none

   make

This will produce the ``microovn.snap`` file that can be then used to install
MicroOVN on your system.

Install MicroOVN
----------------

Using the ``microovn.snap`` file created in the previous section, you can
install MicroOVN in this way:

.. code-block:: none

   sudo snap install --dangerous ./microovn.snap

.. note::

   If you are building latest MicroOVN from the ``main`` branch, it's possible
   that it's using a non-stable core snap as its base. In that case, you may
   get a message like this:

   .. code-block:: none

      Ensure prerequisites for "microovn" are available (cannot install snap base "core24": no snap revision available as specified)

   In such a case, you will need to install the required core snap manually
   from the ``edge`` risk level. For example:

   .. code-block:: none

      snap install core24 --edge

   Then repeat the installation step.

You will also need to manually connect required plugs, as ``snapd`` won't
do it automatically for locally installed snaps.

.. code-block:: none

   for plug in firewall-control \
                hardware-observe \
                hugepages-control \
                network-control \
                openvswitch-support \
                process-control \
                system-trace \
                network-setup-control; do \
       sudo snap connect microovn:$plug;done

To verify that all the required plugs are correctly connected to their slots,
you can run:

.. code-block:: none

   snap connections microovn

An example of correctly connected connected plugs would look like this:

.. code-block:: none

   Interface            Plug                          Slot                       Notes
   content              -                             microovn:ovn-certificates  -
   content              -                             microovn:ovn-chassis       -
   content              -                             microovn:ovn-env           -
   firewall-control     microovn:firewall-control     :firewall-control          manual
   hardware-observe     microovn:hardware-observe     :hardware-observe          manual
   hugepages-control    microovn:hugepages-control    :hugepages-control         manual
   microovn             -                             microovn:microovn          -
   network              microovn:network              :network                   -
   network-bind         microovn:network-bind         :network-bind              -
   network-control      microovn:network-control      :network-control           manual
   openvswitch-support  microovn:openvswitch-support  :openvswitch-support       manual
   process-control      microovn:process-control      :process-control           manual
   system-trace         microovn:system-trace         :system-trace              manual

And if the plugs are not connected, the output would look like this:

.. code-block:: none

   Interface            Plug                          Slot                       Notes
   content              -                             microovn:ovn-certificates  -
   content              -                             microovn:ovn-chassis       -
   content              -                             microovn:ovn-env           -
   firewall-control     microovn:firewall-control     -                          -
   hardware-observe     microovn:hardware-observe     -                          -
   hugepages-control    microovn:hugepages-control    -                          -
   microovn             -                             microovn:microovn          -
   network              microovn:network              :network                   -
   network-bind         microovn:network-bind         :network-bind              -
   network-control      microovn:network-control      -                          -
   openvswitch-support  microovn:openvswitch-support  -                          -
   process-control      microovn:process-control      -                          -
   system-trace         microovn:system-trace         -                          -

.. LINKS
.. _LXD getting started guides: https://documentation.ubuntu.com/lxd/en/latest/getting_started/
