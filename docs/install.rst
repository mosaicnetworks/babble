.. _install:

Install
=======

From Source
^^^^^^^^^^^

Clone the `repository <https://github.com/mosaicnetworks/babble>`__ in the appropriate GOPATH subdirectory:

::

    $ mkdir -p $GOPATH/src/github.com/mosaicnetworks/
    $ cd $GOPATH/src/github.com/mosaicnetworks
    [...]/mosaicnetworks$ git clone https://github.com/mosaicnetworks/babble.git


The easiest way to build binaries is to do so in a hermetic Docker container. 
Use this simple command:  

::

	[...]/babble$ make dist

This will launch the build in a Docker container and write all the artifacts in  
the build/ folder.  

::
	
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
and a CLI tool for managing Go code. Go is very opinionated  and will require 
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

    ok      github.com/mosaicnetworks/babble/net      0.052s
    ok      github.com/mosaicnetworks/babble/common   0.011s
    ?       github.com/mosaicnetworks/babble/cmd      [no test files]
    ?       github.com/mosaicnetworks/babble/cmd/dummy_client [no test files]
    ok      github.com/mosaicnetworks/babble/hashgraph        0.174s
    ok      github.com/mosaicnetworks/babble/node     1.699s
    ok      github.com/mosaicnetworks/babble/proxy    0.018s
    ok      github.com/mosaicnetworks/babble/crypto   0.028s
