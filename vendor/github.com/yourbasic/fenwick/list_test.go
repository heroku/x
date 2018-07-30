package fenwick

import (
	"testing"
)

func TestGet(t *testing.T) {
	n := 100
	a := make([]int64, n)
	for i := range a {
		a[i] = int64(i)
	}
	l := New(a...)
	for i := range a {
		if a[i] != l.Get(i) {
			t.Errorf("Get(%d) = %d; want %d", i, l.Get(i), a[i])
		}
	}
}

func TestSet(t *testing.T) {
	n := 100
	a := make([]int64, n)
	for i := range a {
		a[i] = int64(i)
	}
	l := New(a...)
	for i := range a {
		l.Set(i, 100)
		if l.Get(i) != 100 {
			t.Errorf("Get(%d) = %d; want %d", i, l.Get(i), 100)
		}
	}
}

func TestAdd(t *testing.T) {
	n := 100
	a := make([]int64, n)
	for i := range a {
		a[i] = int64(i)
	}
	l := New(a...)
	for i := range a {
		l.Add(i, 100)
		if l.Get(i) != a[i]+100 {
			t.Errorf("Get(%d) = %d; want %d", i, l.Get(i), a[i]+100)
		}
	}
}

func TestSum(t *testing.T) {
	n := 100
	a := make([]int64, n)
	for i := range a {
		a[i] = int64(i)
	}
	l := New(a...)
	for i := range a {
		var res int64
		for j := 0; j < i; j++ {
			res += a[j]
		}
		if l.Sum(i) != res {
			t.Errorf("Sum(%d) = %d; want %d", i, l.Get(i), res)
		}
	}
}

func TestSumRange(t *testing.T) {
	n := 100
	a := make([]int64, n)
	for i := range a {
		a[i] = int64(i)
	}
	l := New(a...)
	for i := range a {
		for j := i; j < n; j++ {
			var res int64
			for k := i; k < j; k++ {
				res += a[k]
			}
			if l.SumRange(i, j) != res {
				t.Errorf("SumRange(%d, %d) = %d; want %d", i, j, l.SumRange(i, j), res)
			}
		}
	}
}

func TestAppend(t *testing.T) {
	n := 100
	a := make([]int64, n)
	l := new(List)
	if l.Len() != 0 {
		t.Errorf("Len() = %d; want %d", l.Len(), 0)
	}
	for i := range a {
		a[i] = int64(i)
		l.Append(int64(i))
	}
	if l.Len() != n {
		t.Errorf("Len() = %d; want %d", l.Len(), n)
	}
	for i := range a {
		var res int64
		for j := 0; j < i; j++ {
			res += a[j]
		}
		if l.Get(i) != a[i] {
			t.Errorf("Get(%d) = %d; want %d", i, l.Get(i), a[i])
		}
		if l.Sum(i) != res {
			t.Errorf("Sum(%d) = %d; want %d", i, l.Sum(i), res)
		}
	}
}
