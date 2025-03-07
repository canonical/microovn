==============
Accessing logs
==============

The :ref:`MicroOVN services` provide logs as part of their normal operation.

By default they are provided through the systemd journal, and can be accessed
through the use of the ``journalctl`` or ``snap logs`` commands.

This is how you can access the logs of the ``microovn.chassis`` service using
the ``snap logs`` command:

.. code-block:: none

   snap logs microovn.chassis

and using the ``journalctl`` command:

.. code-block:: none

   journalctl -u snap.microovn.chassis

This is how you can view a live log display for the same service using
the ``snap logs`` command:

.. code-block:: none

   snap logs -f microovn.chassis

and using the ``journalctl`` command:

.. code-block:: none

   journalctl -f -u snap.microovn.chassis

Log files
---------

Inside the ``/var/snap/microovn/common/logs`` directory you will find files for
each individual service, however these will either be empty or not contain
updated information, this is intentional.

On a fresh install the files are created, as a precaution, in the event a need
arises for enabling `debug logging`_.  When upgrading MicroOVN, existing files
will be retained, but not updated.

Debug logging
-------------

The Open vSwitch (OVS) and Open Virtual Network (OVN) daemons have a rich set
of debug features, one of which is the ability to specify log levels for
individual modules at run time.

A list of modules can be acquired through the ``ovs-appctl`` and
``ovn-appctl`` commands.

This is how to enable debug logging for the Open vSwitch ``vswitchd`` module:

.. code-block:: none

   ovs-appctl vlog/set vswitchd:file:dbg

This is how to enable debug logging for the Open Virtual Network ``reconnect``
module:

.. code-block:: none

   ovn-appctl vlog/set reconnect:file:dbg

For more details on how to configure logging, see `ovs-appctl manpage`_.

.. LINKS
.. _ovs-appctl manpage: https://docs.openvswitch.org/en/latest/ref/ovs-appctl.8/#logging-commands
