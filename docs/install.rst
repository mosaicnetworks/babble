Install
=======

Go
^^

Babble is written in `Golang <https://golang.org/>`__. Hence, the first step is to install  
(**Go version 1.9 or above**) which is both the programming language  
and a CLI tool for managing Go code. Go is very opinionated  and will require you to  
`define a workspace <https://golang.org/doc/code.html#Workspaces>`__ where all your go code will reside. 

Babble and dependencies  
^^^^^^^^^^^^^^^^^^^^^^^

Clone the `repository <https://bitbucket.org/mosaicnet/babble>`__ in the appropriate GOPATH subdirectory:

::

    $ mkdir -p $GOPATH/src/bitbucket.org/mosaicnet/
    $ cd $GOPATH/src/bitbucket.org/mosaicnet
    [...]/mosaicnet$ git clone https://bitbucket.org/mosaicnet/babble.git

Babble uses `Glide <http://github.com/Masterminds/glide>`__ to manage dependencies.

::

    [...]/babble$ sudo add-apt-repository ppa:masterminds/glide && sudo apt-get update
    [...]/babble$ sudo apt-get install glide
    [...]/babble$ glide install

This will download all dependencies and put them in the **vendor** folder.

Testing
^^^^^^^

Babble has extensive unit-testing. Use the Go tool to run tests:  

::

    [...]/babble$ make test

If everything goes well, it should output something along these lines:  

::

    ok      bitbucket.org/mosaicnet/babble/net      0.052s
    ok      bitbucket.org/mosaicnet/babble/common   0.011s
    ?       bitbucket.org/mosaicnet/babble/cmd      [no test files]
    ?       bitbucket.org/mosaicnet/babble/cmd/dummy_client [no test files]
    ok      bitbucket.org/mosaicnet/babble/hashgraph        0.174s
    ok      bitbucket.org/mosaicnet/babble/node     1.699s
    ok      bitbucket.org/mosaicnet/babble/proxy    0.018s
    ok      bitbucket.org/mosaicnet/babble/crypto   0.028s
