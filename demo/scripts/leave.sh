#!/bin/bash

NODE=${1:-1}

docker kill --signal=SIGINT node$NODE client$NODE