package hashgraph

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestNewBadgerStore(t *testing.T) {
	dir, err := ioutil.TempDir("test_data", "badger")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	participants := map[string]int{
		"alice":   0,
		"bob":     1,
		"charlie": 2,
	}
	cacheSize := 100

	store, err := NewBadgerStore(participants, cacheSize, dir)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if store.path != dir {
		t.Fatalf("unexpected path %q", store.path)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
