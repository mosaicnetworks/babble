# Docker 

This folder contains the scripts to build and push various Docker images.

All these images are publicly available on 
[Docker Hub](https://hub.docker.com/u/mosaicnetworks/) so it is not necessary  
to build them in order to actually use them. 

## Mosaic Networks Developers

Always make sure to update the versions in the makefile.

## Babble image

Simple wrapper around Babble based on alpine linux.  
Refer to demo/ for usage.

## Dummy image

Simple wrapper around the Dummy Client application for Babble. This is effectively  
a group chat application.  
Refer to demo/ for usage.

## Watcher image

This image describes a container that monitors the consensus stats for a Babble 
docker network. It is based on the watch.sh script.

## Mobile Image 

The mobile image contains all the tools needed to build a mobile babble library.


## Glider Image

The glider image contains all the tools to compile Babble for almost any platform.  

We use it in two places:

1) In the circleci job to automate building and testing Babble everytime we push  
   to Github.
2) In the dist_build.sh script to do a hermetic build and generate binaries for  
   many platforms.
