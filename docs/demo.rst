.. _demo:

EVM Demo
========

One of the coolest things that software like Babble can do is to make many 
computers behave as one. This is basically what Bitcoin and Ethereum do but they 
work with their own consensus algorithms which sacrifice speed and efficiency 
over the ability to run in a public environment.

Here we will demonstrate how to run the Ethereum Virtual Machine (EVM) on top of 
Babble. We took the Go-Ethereum codebase and stripped out all the pieces we 
didn't need. Then we plugged that into Babble and showed how the result can be 
used as a distributed computing engine for permissioned networks with the 
maximum level of Byzantine Fault Tolerance and greater speed.

::

            
                    =============================================
    ============    =  ===============         ===============  =       
    =          =    =  = Service     =         = State App   =  =
    =  Client  <-----> =             = <------ =             =  =
    =          =    =  = -API        =         = -EVM        =  =
    ============    =  = -Keystore   =         = -Trie       =  =
                    =  =             =         = -Database   =  =
                    =  ===============         ===============  =
                    =         |                       |         =
                    =  =======================================  =
                    =  = Babble Proxy                        =  =
                    =  =                                     =  =
                    =  =======================================  =
                    =         |                       ^         =  
                    ==========|=======================|==========
                              |Txs                    |Blocks
                    ==========|=======================|==========
                    = Babble  v                       |         =
                    =                                           =                                             
                    =                   ^                       =
                    ====================|========================  
                                        |
                                        |
                                        v
                                    Consensus
    

First things first
------------------

The code for evm-babble lives in its own repo. Let us download it and give a 
quick overview of what it does.
  
Clone the `repository <https://github.com/mosaicnetworks/evm-babble>`__ in the 
appropriate GOPATH subdirectory:

::

    $ cd $GOPATH/src/github.com/mosaicnetworks
    [...]/mosaicnetworks$ git clone https://github.com/mosaicnetworks/evm-babble.git

Then, follow the instructions in the README to install dependencies and complete 
the installation.

The EVM is a virtual machine specifically designed to run untrusted code on a 
network of computers. Every transaction applied to the EVM modifies the State 
which is persisted in a Merkle Patricia tree. This data structure allows to 
simply check if a given transaction was actually applied to the VM and can 
reduce the entire State to a single hash (merkle root) rather analogous to a 
fingerprint.

The EVM is meant be used in conjunction with a system that broadcasts 
transactions across network participants and ensures that everyone executes the 
same transactions in the same order. Ethereum uses a Blockchain and a Proof of 
Work consensus algorithm. Here, we use Babble. 

Crowd Funding Example
---------------------

The /demo folder of the evm-babble repo contains some scripts and instructions 
to setup a testnet and walk through an example where the user will run a secure 
decentralized crowd-funding campaign on a permissioned Babble network. It shows 
how to deploy a SmartContract that coordinates financial contributions and pays 
off the beneficiary when the funding target is reached. The SmartContract in 
question was only created for the purposes of this demo so please do not use it 
for a real crowd-funding campaign. We recommend opening multiple terminals in 
parallel to monitor the consensus stats and logs on the various nodes while the 
demo runs. This will help to get an understanding of all the work that is 
happening in the background to securely replicate all commands on multiple nodes 
for greater security.
