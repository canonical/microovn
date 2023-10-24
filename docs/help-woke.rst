:orphan:

===================================
Inclusive language check exemptions
===================================

This page provides an overview of two inclusive language check exemption
methods for files written in ReST format. See the `woke documentation`_ for
full coverage.

Exempt a word
-------------

To exempt an individual word, place a comment on a line immediately preceding
the line containing the word in question. This special comment is to include
the syntax ``wokeignore:rule=<SOME_WORD>``. For instance:

.. code-block:: ReST

   .. wokeignore:rule=whitelist
   This is your text. The word in question is here: whitelist. More text.

Here is an example of an exemption that acts upon an element of a URL that is
expressed using the link definition method (typically at the bottom of a file):

.. code-block:: ReST

   .. LINKS
   .. wokeignore:rule=master
   .. _link definition: https://some-external-site.io/master/some-page.html

Exempt an entire file
---------------------

A more drastic solution is to make an exemption for the contents of an entire
file.

Start by placing file ``.wokeignore`` into your project's root directory. Then,
to exempt file ``docs/foo/bar.rst``, add the following line to it:

.. code-block:: none

   foo/bar.rst

.. LINKS
.. _woke documentation: https://docs.getwoke.tech/ignore
