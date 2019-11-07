# Making a Mobile Library Release

From a current version of the babble repo in the appropriate branch (currently it needs to be mobile):


First we make the mobile docker image. This stage does not include babble code, so should not be necessary besides the first run and when we wish to update go / gomobile / Android SDK. This is slow and involves Gigabytes of downloads - the Android SDK and NDK are heavy. 

```bash
[...babble]$ cd docker
[...babble]$ make mobile-image
[...babble]$ cd ..

```


```bash
[...babble]$ make mobile-dist
[...babble]$ ls -l build/distmobile
-rw-rw-r-- 1 jon jon 37815739 Nov  7 12:43 babble_0.5.9_android_library.zip
-rw-r--r-- 1 jon jon 38262258 Nov  7 12:43 babble_0.5.9_mobile.aar
-rw-r--r-- 1 jon jon     8532 Nov  7 12:43 babble_0.5.9_mobile-sources.jar
-rw-rw-r-- 1 jon jon      192 Nov  7 12:43 babble_0.5.9_SHA256SUMS
```
babble_0.5.9_android_library.zip is the release to upload.



