## Build from source

### Dependencies
Java JDK, Android NDK, Go mobile tools

```bash
$ go get golang.org/x/mobile/cmd/gomobile
$ gomobile init -ndk ~/PATH/TO/ANDROID/NDK
```

### Building android library
To compile Go package as android library execute
```bash
$ gomobile bind -v -target=android -tags=mobile github.com/mosaicnetworks/babble/src/mobile
```

## Import the Babble Module

Follow Oliver's answer:   
https://stackoverflow.com/questions/16682847/how-to-manually-include-external-aar-package-using-new-gradle-android-build-syst

Sometimes, Android Studio says “cannot resolve symbol” even if the project 
compiles. In this case, do the following:

*"File" -> "Invalidate Caches..." -> "Invalidate and Restart"*
