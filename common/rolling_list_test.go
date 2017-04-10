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
package common

import (
	"fmt"
	"testing"
)

func TestRollingWindow(t *testing.T) {
	size := 10
	testSize := 3 * size
	rollingList := NewRollingList(size)
	items := []string{}
	for i := 0; i < testSize; i++ {
		item := fmt.Sprintf("item%d", i)
		rollingList.Add(item)
		items = append(items, item)
	}
	cached, ts := rollingList.Get()

	if ts != testSize {
		t.Fatalf("tot should be %d, not %d", testSize, ts)
	}

	start := (testSize / (2 * size)) * (size)
	count := testSize - start

	for i := 0; i < count; i++ {
		if cached[i] != items[start+i] {
			t.Fatalf("cached[%d] should be %s, not %s", i, items[start+i], cached[i])
		}
	}
}
