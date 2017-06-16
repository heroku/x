package cache

import "testing"

func TestHashImplementsStorage(t *testing.T) {
	var i interface{} = Hash{}
	if _, ok := i.(Storage); !ok {
		t.Fatalf("type Hash does not implement interface Storage")
	}
}

func TestVolatileImplementsStorage(t *testing.T) {
	var i interface{} = Volatile{}
	if _, ok := i.(Storage); !ok {
		t.Fatalf("type Volatile does not implement interface Storage")
	}
}
