:orphan:

==============================
Working with the documentation
==============================

**Note:** This page is for documentation contributors. It assumes that this
repository has been enabled for doc builds as described in file ``setup.rst``.

There are make targets defined in the documentation ``Makefile`` file that set
up your local environment and allow you to view the documentation. You may also
need to occasionally change the build configuration as new content is added.

Set up your local environment
-----------------------------

Use the ``install`` make target to set up your local environment:

.. code-block:: none

   make install

This will create a virtual environment (``.sphinx/venv``) and install
dependency software (``.sphinx/requirements.txt``) within it.

**Note**: The starter pack uses the latest compatible version of all tools and
does not pin its requirements. This might change temporarily if there is an
incompatibility with a new tool version. There is therefore no need in using a
tool like Renovate to automatically update the requirements.

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

Local checks
------------

Ensure the following local checks run error-free prior to submitting a change
(PR).

Local build
~~~~~~~~~~~

Run a clean build:

.. code-block:: none

   make clean-doc
   make html

Spelling check
~~~~~~~~~~~~~~

Ensure that there are no spelling mistakes:

.. code-block:: none

   make spelling

Inclusive language check
~~~~~~~~~~~~~~~~~~~~~~~~

Perform a check for non-inclusive language:

.. code-block:: none

   make woke

Link check
~~~~~~~~~~

Validate hyperlinks:

.. code-block:: none

   make linkcheck

Change the build configuration
------------------------------

Occasionally you may want, or need, to alter the build configuration.

False-positive misspellings
~~~~~~~~~~~~~~~~~~~~~~~~~~~

To add exceptions for words the spellcheck incorrectly marks as wrong, edit the
``.custom_wordlist.txt`` file.

File ``.wordlist.txt`` should not be touched since it is maintained centrally.
It contains words that apply across all projects.

Unwanted link checks
~~~~~~~~~~~~~~~~~~~~

To prevent links from being validated, edit the ``linkcheck_ignore`` variable
in the ``conf.py`` file. Example reasons for doing this include:

* the links are local
* the validation of the links causes errors for no good reason

HTML redirects
~~~~~~~~~~~~~~

HTML redirects can be added to ensure that old links continue to work when you
move files around. To do so, specify the old and new paths in the ``redirects``
setting in file ``custom_conf.py``.

Customisation of inclusive language checks
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

By default, the inclusive language check is applied only to reST files located
under the documentation directory (usually ``docs``). To check Markdown files,
for example, or to use a location other than the ``docs`` sub-tree, you must
change how the ``woke`` tool is invoked from within ``docs/Makefile`` (see
the `woke User Guide <https://docs.getwoke.tech/usage/#file-globs>`_ for help).

Some circumstances may compel you to retain some non-inclusive words. In such
cases you will need to create check exemptions for them. See file
:doc:`help-woke` for how to do that.
