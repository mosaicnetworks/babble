.. _install:

Install
=======

From Source
^^^^^^^^^^^

Clone the `repository <https://github.com/mosaicnetworks/babble>`__ in the appropriate GOPATH subdirectory:

.. code:: bash

    $ mkdir -p $GOPATH/src/github.com/mosaicnetworks/
    $ cd $GOPATH/src/github.com/mosaicnetworks
    [...]/mosaicnetworks$ git clone https://github.com/mosaicnetworks/babble.git


The easiest way to build binaries is to do so in a hermetic Docker container.
Use this simple command:

.. code:: bash

    [...]/babble$ make dist

This will launch the build in a Docker container and write all the artifacts in
the build/ folder.

.. code:: bash

    [...]/babble$ tree build
    build/
    ├── dist
    │   ├── babble_0.1.0_darwin_386.zip
    │   ├── babble_0.1.0_darwin_amd64.zip
    │   ├── babble_0.1.0_freebsd_386.zip
    │   ├── babble_0.1.0_freebsd_arm.zip
    │   ├── babble_0.1.0_linux_386.zip
    │   ├── babble_0.1.0_linux_amd64.zip
    │   ├── babble_0.1.0_linux_arm.zip
    │   ├── babble_0.1.0_SHA256SUMS
    │   ├── babble_0.1.0_windows_386.zip
    │   └── babble_0.1.0_windows_amd64.zip
    └── pkg
        ├── darwin_386
        │   └── babble
        ├── darwin_amd64
        │   └── babble
        ├── freebsd_386
        │   └── babble
        ├── freebsd_arm
        │   └── babble
        ├── linux_386
        │   └── babble
        ├── linux_amd64
        │   └── babble
        ├── linux_arm
        │   └── babble
        ├── windows_386
        │   └── babble.exe
        └── windows_amd64
            └── babble.exe

Go Devs
^^^^^^^

Babble is written in `Golang <https://golang.org/>`__. Hence, the first step is
to install **Go version 1.9 or above** which is both the programming language
and a CLI tool for managing Go code. Go is very opinionated and will require
you to `define a workspace <https://golang.org/doc/code.html#Workspaces>`__
where all your go code will reside.

Dependencies
^^^^^^^^^^^^

Babble uses `Glide <http://github.com/Masterminds/glide>`__ to manage
dependencies. For Ubuntu users:

::

    [...]/babble$ curl https://glide.sh/get | sh
    [...]/babble$ glide install

This will download all dependencies and put them in the **vendor** folder.

Testing
^^^^^^^

Babble has extensive unit-testing. Use the Go tool to run tests:

::

    [...]/babble$ make test

If everything goes well, it should output something along these lines:

::

    ?       github.com/mosaicnetworks/babble/src/babble     [no test files]
    ok      github.com/mosaicnetworks/babble/src/common     0.015s
    ok      github.com/mosaicnetworks/babble/src/crypto     0.122s
    ok      github.com/mosaicnetworks/babble/src/hashgraph  10.270s
    ?       github.com/mosaicnetworks/babble/src/mobile     [no test files]
    ok      github.com/mosaicnetworks/babble/src/net        0.012s
    ok      github.com/mosaicnetworks/babble/src/node       19.171s
    ok      github.com/mosaicnetworks/babble/src/peers      0.038s
    ?       github.com/mosaicnetworks/babble/src/proxy      [no test files]
    ok      github.com/mosaicnetworks/babble/src/proxy/dummy        0.013s
    ok      github.com/mosaicnetworks/babble/src/proxy/inmem        0.037s
    ok      github.com/mosaicnetworks/babble/src/proxy/socket       0.009s
    ?       github.com/mosaicnetworks/babble/src/proxy/socket/app   [no test files]
    ?       github.com/mosaicnetworks/babble/src/proxy/socket/babble        [no test files]
    ?       github.com/mosaicnetworks/babble/src/service    [no test files]
    ?       github.com/mosaicnetworks/babble/src/version    [no test files]
    ?       github.com/mosaicnetworks/babble/cmd/babble     [no test files]
    ?       github.com/mosaicnetworks/babble/cmd/babble/commands    [no test files]
    ?       github.com/mosaicnetworks/babble/cmd/dummy      [no test files]
    ?       github.com/mosaicnetworks/babble/cmd/dummy/commands     [no test files]

