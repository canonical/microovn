=================
Automatic Aliases
=================

MicroOVN is distributed by snap, which has automatic aliases for OVN and OVS
binaries. You can view these with the snap aliases command:

.. code-block:: none

    snap aliases microovn

.. code-block:: none

	Command                Alias         Notes
	microovn.ovn-appctl    ovn-appctl    -
	microovn.ovn-nbctl     ovn-nbctl     -
	microovn.ovn-sbctl     ovn-sbctl     -
	microovn.ovn-trace     ovn-trace     -
	microovn.ovs-appctl    ovs-appctl    -
	microovn.ovs-dpctl     ovs-dpctl     -
	microovn.ovs-ofctl     ovs-ofctl     -
	microovn.ovs-vsctl     ovs-vsctl     -
	microovn.ovsdb-client  ovsdb-client  -
	microovn.ovsdb-tool    ovsdb-tool    -

Further inspection can be done by inspecting the files themselves:

.. code-block:: none

	ls $(which ovn-nbctl) -l

.. code-block:: none

	lrwxrwxrwx 1 root root 18 Nov 28 15:25 /snap/bin/ovn-nbctl -> microovn.ovn-nbctl

These aliases are not related to the MicroOVN snap version and are managed by
the store. All installations, done through the snap store, should have access to these aliases. This does mean if you install a locally built version of MicroOVN, these aliases are not created for you.
