# Release Procedures

## Version Number Checks

The mobile release of `babble` (this repo) and the `babble-android` release 
should use the same version of the Android SDK. 

In the `babble` repo, `/docker/mobile/Dockerfile` contains the following 
sections defining Android SDK and GO versions:

```
ENV SDK_URL="https://dl.google.com/android/repository/sdk-tools-linux-4333796.zip" \
    ANDROID_HOME="/usr/local/android-sdk" \
    ANDROID_VERSION=29 \
    ANDROID_BUILD_TOOLS_VERSION=29.0.3
```

```
ENV GOLANG_VERSION 1.13.7
```

Latest version numbers are shown on the pages below:

* [Android SDK Build Tools versions](https://developer.android.com/studio/releases/build-tools) 
* [Go Versions](https://golang.org/dl/)

If changing the go version, you will need to cut and paste the checksum into 
approximately line 70 of the Dockerfile.


In the ``babble-android`` repo in ``/babble/build.gradle`` is a section as 
follows:

```
android {
    compileSdkVersion 29

    defaultConfig {
        minSdkVersion 19
        targetSdkVersion 29
```

Make sure the ``compileSdkVersion`` and ``targetSdkVersion`` match the versions 
in the ``babble`` repo.

In Android Studio, Open Tools/SDK Manager. Click the SDK Tools, and click the 
"Show package details" check box.  Check the version of the Android SDK Build
tools installed, matches the version from above from the `babble` repo 
Dockerfile.

## Release Procedure for Babble Mobile

In the root of the babble repo, pull the latest released master branch from 
github. 

```bash
$ cd docker
$ make mobile-image
$ cd ..
$ make mobile-dist
```

By default, this will name the release 
``babble_<branch name>_ <commit-hash>_android_library.zip``. To give it a
different name, use the ``VERSION`` parameter. 

Ex:

```bash
make mobile-dist VERSION=testing
```

This will produce ``babble_testing_android_library.zip``.

The files are written to ``./build/distmobile/``.

When creating a new release of Babble from the master branch, use 
``VERSION=v0.6.2`` (for example) and upload 
``/build/distmobile/babble_v0.6.2_android_library.zip`` to the Babble Github 
repo releases page. 

