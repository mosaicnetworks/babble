# Glider Image

The glider image contains all the tools to compile Babble for almost any platform.  

We use it in two places:

1) In the circleci job to automate building and testing Babble everytime we push  
   to Github.
2) In the dist_build.sh script to do a hermetic build and generate binaries for  
   many platforms.

We should keep public versioned copies of this image on Docker Hub. There is an  
organisation setup on Docker Hub for Mosaic Networks and a [public repo for glider](https://hub.docker.com/r/mosaicnetworks/glider/)

Update the version number if changes are made to the Dockerfile

To build the image:

```bash
docker build -t mosaicnetworks/glider:<version> .
```

Push to Docker Hub:

```bash
docker login
docker push mosaicnetworks/glider:<version>
```
