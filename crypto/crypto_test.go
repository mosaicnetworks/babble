/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package crypto

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestPem(t *testing.T) {
	// Create a test dir
	dir, err := ioutil.TempDir("", "swirld")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}
	defer os.RemoveAll(dir)

	// Create the PEM key
	pemKey := NewPemKey(dir)

	// Try a read, should get nothing
	key, err := pemKey.ReadKey()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if key != nil {
		t.Fatalf("key is not nil")
	}

	// Initialize a key
	key, _ = GenerateECDSAKey()
	if err := pemKey.WriteKey(key); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Try a read, should get key
	nKey, err := pemKey.ReadKey()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(*nKey, *key) {
		t.Fatalf("Keys do not match")
	}
}
