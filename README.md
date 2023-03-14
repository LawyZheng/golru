# GoLRU Cache [![Build Status](https://travis-ci.com/orcaman/concurrent-map.svg?branch=master)](https://travis-ci.com/orcaman/concurrent-map)

A light, high performance LRU cache package for Go, based on [Concurrent-Map](http://github.com/orcaman/concurrent-map). This package have features as followed:
- [X] nearly LRU cache algorithm
- [X] use generic design，no need to serialize the struct or assert the type of interface.


## usage

Import the package:

```go
import (
	"github.com/lawyzheng/golru"
)

```

```bash
go get "github.com/lawyzheng/golru"
```

## example

```go

	// Create a new map.
	c := golru.New[string](0)

	// Sets item within map, sets "bar" under key "foo"
	c.Set("foo", "bar")

	// Retrieve item from map.
	bar, ok := c.Get("foo")

	// Removes item under key "foo"
	c.Remove("foo")

```

For more examples have a look at lru_test.go.

Running tests:

```bash
go test "github.com/lawyzheng/golru"
```

## guidelines for contributing

Contributions are highly welcome. In order for a contribution to be merged, please follow these guidelines:
- Open an issue and describe what you are after (fixing a bug, adding an enhancement, etc.).
- According to the core team's feedback on the above mentioned issue, submit a pull request, describing the changes and linking to the issue.
- New code must have test coverage.
- If the code is about performance issues, you must include benchmarks in the process (either in the issue or in the PR).
- In general, we would like to keep `golru` as simple as possible. Please keep this in mind when opening issues.

## language
- [中文说明](./README-zh.md)

## appreciation

Thanks for [orcaman/concurrent-map](http://github.com/orcaman/concurrent-map) to offer the basic idea.

## license
MIT (see [LICENSE](https://github.com/orcaman/concurrent-map/blob/master/LICENSE) file)
