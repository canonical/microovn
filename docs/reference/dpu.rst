===============
DPU Integration
===============

MicroOVN detects if it is running on the DPU side of the PCI complex as part of
bootstrap/join. It uses devlink to find a valid port representing host facing resources, valid
ports are either ``"pcivf"`` flavour ports or ``"pcipf"`` flavour ports with the
controller value being 1. This is due to the fact that local DPU side controller
ports will have the controller value as 1 according to the
`Devlink Port Documentation`, but this does not guarantee it is a local
controller port on the DPU side, which is why we also check the port flavour.

.. note::

   For further understanding on our reasoning, check out:

   - `Devlink Controller Commit`
   - `Devlink Port Code`


We then use lspci to extract the serial number for the given port
(identified by its PCI address). This is then inserted into the
card-serial-number key in the external-ids:ovn-cms-options dictionary.

This serial number acts as a bridge for discovery and coordination.
Allowing the networking control plane to match the hypervisor host's VF
with the correct DPU, despite them running separate operating
systems with different hostnames, in order to properly handle port binding

.. LINKS
.. _`Devlink Port Documentation`: https://docs.kernel.org/networking/devlink/devlink-port.html
.. _`Devlink Controller Commit`: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=3a2d9588c4f79adae6a0e986b64ebdd5b38085c6
.. _`Devlink Port Code`: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/net/core/devlink.c?id=cd76dcd68d96aa5bbc63b7ef25a87a1dbea3d73c#n1185
