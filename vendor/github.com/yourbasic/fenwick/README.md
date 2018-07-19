# Your basic Fenwick tree [![GoDoc](https://godoc.org/github.com/yourbasic/fenwick?status.svg)][godoc-fenwick]

### Go list data structure supporting prefix sums

A Fenwick tree, or binary indexed tree, is a space-efficient
list data structure that can efficiently update elements and
calculate prefix sums in a list of numbers.

![Checklist](res/checklist.jpg)

### Installation

Once you have [installed Go][golang-install], run this command
to install the `fenwick` package:

    go get github.com/yourbasic/fenwick
    
### Documentation

There is an online reference for the package at
[godoc.org/github.com/yourbasic/fenwick][godoc-fenwick].

### Roadmap

* The API of this library is frozen.
* Version numbers adhere to [semantic versioning][sv].

The only accepted reason to modify the API of this package
is to handle issues that can't be resolved in any other
reasonable way.

Stefan Nilsson – [korthaj](https://github.com/korthaj)

[godoc-fenwick]: https://godoc.org/github.com/yourbasic/fenwick
[golang-install]: http://golang.org/doc/install.html
[sv]: http://semver.org/
