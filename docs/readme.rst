:orphan:

==============================
Working with the documentation
==============================

**Important:** This page is for documentation contributors. It assumes that an
admin has set up this repository to work with Read the Docs as described in
file ``setup.rst``.

There are make targets defined in the ``Makefile`` that set up your local
environment and allow you to view the documentation. You may also need to
occasionally change the build configuration as new content is added.

Set up your local environment
-----------------------------

Use the ``install`` make target to set up your local environment:

.. code-block:: none

   make install

This will create a virtual environment (``.sphinx/venv``) and install
dependency software (``.sphinx/requirements.txt``) within it.

A complete set of pinned, known-working dependencies is included in
``.sphinx/pinned-requirements.txt``.

View the documentation
----------------------

Use the ``run`` make target to view the documentation:

.. code-block:: none

   make run

This will do several things:

* activate the virtual environment
* build the documentation
* serve the documentation on **127.0.0.1:8000**
* rebuild the documentation each time a file is saved
* send a reload page signal to the browser when the documentation is rebuilt

The ``run`` target is therefore very convenient when preparing to submit a
change to the documentation. For a more manual approach, to strictly build and
serve content, explore the ``html`` and ``serve`` make targets, respectively.

Change the build configuration
------------------------------

Occasionally you may want, or need, to alter the build configuration.

False-positive misspellings
~~~~~~~~~~~~~~~~~~~~~~~~~~~

To add exceptions for words the spellcheck incorrectly marks as wrong, edit the
``.wordlist.txt`` file.

Unwanted link checks
~~~~~~~~~~~~~~~~~~~~

To prevent links from being validated, edit the ``linkcheck_ignore`` variable
in the ``conf.py`` file. Example reasons for doing this include:

* the links are local
* the validation of the links causes errors for no good reason
