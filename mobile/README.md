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
$ gomobile bind -v -target=android/arm -tags=mobile github.com/babbleio/babble/mobile
```

Copy the output **aar** file to ```demo/android/babble```
