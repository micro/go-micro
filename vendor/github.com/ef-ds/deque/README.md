# deque [![Build Status](https://travis-ci.com/ef-ds/deque.svg?branch=master)](https://travis-ci.com/ef-ds/deque) [![codecov](https://codecov.io/gh/ef-ds/deque/branch/master/graph/badge.svg)](https://codecov.io/gh/ef-ds/deque) [![Go Report Card](https://goreportcard.com/badge/github.com/ef-ds/deque)](https://goreportcard.com/report/github.com/ef-ds/deque)  [![GoDoc](https://godoc.org/github.com/ef-ds/deque?status.svg)](https://godoc.org/github.com/ef-ds/deque)

Package deque implements a very fast and efficient general purpose queue/stack/deque data structure that is specifically optimized to perform when used by Microservices and serverless services running in production environments. Internally, deque stores the elements in a dynamically growing circular doubly linked list of arrays.


## Install
From a configured [Go environment](https://golang.org/doc/install#testing):
```sh
go get -u github.com/ef-ds/deque
```

If you are using dep:
```sh
dep ensure -add github.com/ef-ds/deque@1.0.4
```

We recommend to target only released versions for production use.


## How to Use
As a First-In-First-Out [queue](https://en.wikipedia.org/wiki/Queue_(abstract_data_type)).

```go
package main

import (
	"fmt"

	"github.com/ef-ds/deque"
)

func main() {
	var d deque.Deque

	for i := 1; i <= 5; i++ {
		d.PushBack(i)
	}
	for d.Len() > 0 {
		v, _ := d.PopFront()
		fmt.Println(v)
	}
}
```

Output:
```
1
2
3
4
5
```


As a Last-In-First-Out [stack](https://en.wikipedia.org/wiki/Stack_(abstract_data_type)).

```go
package main

import (
	"fmt"

	"github.com/ef-ds/deque"
)

func main() {
	var d deque.Deque

	for i := 1; i <= 5; i++ {
		d.PushBack(i)
	}
	for d.Len() > 0 {
		v, _ := d.PopBack()
		fmt.Println(v)
	}
}
```

Output:
```
5
4
3
2
1
```

Also refer to the [integration](integration_test.go) and [API](api_test.go) tests.



## Tests
Besides having 100% code coverage, deque has an extensive set of [unit](unit_test.go), [integration](integration_test.go) and [API](api_test.go) tests covering all happy, sad and edge cases.

Performance and efficiency are major concerns, so deque has an extensive set of benchmark tests as well comparing the deque performance with a variety of high quality open source deque implementations.

See the [benchmark tests](https://github.com/ef-ds/deque-bench-tests/blob/master/BENCHMARK_TESTS.md) for details.

When considering all tests, deque has over 10x more lines of testing code when compared to the actual, functional code.


## Performance
Deque has constant time (O(1)) on all its operations (PushFront/PushBack/PopFront/PopBack/Len). It's not amortized constant as it is with most [traditional growing array deques](https://en.wikipedia.org/wiki/Double-ended_queue#Complexity) because deque never copies more than 16 (maxFirstSliceSize/sliceGrowthFactor) items and when it expands or grow, it never does so by more than 256 (maxInternalSliceSize) items in a single operation.

Deque, either used as a FIFO queue or LIFO stack, offers either the best or very competitive performance across all test sets, suites and ranges.

As a general purpose FIFO deque or LIFO stack, deque offers, by far, the most balanced and consistent performance of all tested data structures.

See [performance](https://github.com/ef-ds/deque-bench-tests/blob/master/PERFORMANCE.md) for details.


## Design
The Efficient Data Structures (ef-ds) deque employs a new, modern deque design: a ring shaped, auto shrinking, linked slices design.

That means the [double-ended queue](https://en.wikipedia.org/wiki/Double-ended_queue) is a [doubly-linked list](https://en.wikipedia.org/wiki/Doubly_linked_list) where each node value is a fixed size [slice](https://tour.golang.org/moretypes/7). It is ring in shape because the linked list is a [circular one](https://en.wikipedia.org/wiki/Circular_buffer), where the last node always points to the first one in the ring. It also auto shrinks as the values are popped off the deque, keeping itself as a lean and low memory footprint data structure even after heavy use.

![ns/op](testdata/deque.jpg?raw=true "Deque Design")


### Design Considerations
Deque uses linked slices as its underlying data structure. The reason for the choice comes from two main observations of slice based deques/queues/stacks:

1. When the deque/queue/stack needs to expand to accommodate new values, [a new, larger slice needs to be allocated](https://en.wikipedia.org/wiki/Dynamic_array#Geometric_expansion_and_amortized_cost) and used
2. Allocating and managing large slices is expensive, especially in an overloaded system with little available physical memory

To help clarify the scenario, below is what happens when a slice based deque that already holds, say 1bi items, needs to expand to accommodate a new item.

Slice based implementation.

- Allocate a new, twice the size of the previous allocated one, say 2 billion positions slice
- Copy over all 1 billion items from the previous slice into the new one
- Add the new value into the first unused position in the new slice, position 1000000001

The same scenario for deque plays out like below.

- Allocate a new 256 size slice
- Set the previous and next pointers
- Add the value into the first position of the new slice, position 0

The decision to use linked slices was also the result of the observation that slices goes to great length to provide predictive, indexed positions. A hash table, for instance, absolutely need this property, but not a deque. So deque completely gives up this property and focus on what really matters: add and retrieve from the edges (front/back). No copying around and repositioning of elements is needed for that. So when a slice goes to great length to provide that functionality, the whole work of allocating new arrays, copying data around is all wasted work. None of that is necessary. And this work costs dearly for large data sets as observed in the tests.

While linked slices design solve the slice expansion problem very effectively, it doesn't help with many real world usage scenarios such as in a stable processing environment where small amount of items are pushed and popped from the deque in a sequential way. This is a very common scenario for [Microservices](https://en.wikipedia.org/wiki/Microservices) and [serverless](https://en.wikipedia.org/wiki/Serverless_computing) services, for instance, where the service is able to handle the current traffic without stress.

To address the stable scenario in an effective way, deque keeps its internal linked arrays in a circular, ring shape. This way when items are pushed to the deque after some of them have been removed, the deque will automatically move over its tail slice back to the old head of the deque, effectively reusing the same already allocated slice. The result is a deque that will run through its ring reusing the ring to store the new values, instead of allocating new slices for the new values.

Another important real world scenario is when a service is subject to high and low traffic patterns over time. It is desired the service to be able to scale out quickly and effectively to handle the extra traffic when needed, but also to scale in, decreasing resource usage, such as CPU and memory, when the traffic subsides, so the cost to run the system over time is optimized.

To solve this problem, deque employs an automatically shrinking mechanism that will shrink, releasing the extra resources from memory, as the number of items in the deque decreases. This mechanism allows the queue to shrink when needed, keeping itself lean, but also solves a major problem of most ring based implementations: once inflated, the data structure either never shrinks or require manual "shrink" calls, which can be tricky to use as the ideal, most optimized moment to shrink is not always clear and well defined.

Having said that, a data structure that completely shrinks after use, when it is used again, it means it has to expand again to accommodate the new values, hindering performance on refill scenarios (where a number of items is added and removed from the deque successively). To address this scenario, deque keeps a configurable number of internal, empty, slices in its ring. This way in refill scenarios deque is able to scale out very quickly, but still managing to keep the memory footprint very low.



## Supported Data Types
Similarly to Go's standard library list, [list](https://github.com/golang/go/tree/master/src/container/list),
[ring](https://github.com/golang/go/tree/master/src/container/ring) and [heap](https://github.com/golang/go/blob/master/src/container/heap/heap.go) packages, deque supports "interface{}" as its data type. This means it can be used with any Go data types, including int, float, string and any user defined structs and pointers to interfaces.

The data types pushed into the deque can even be mixed, meaning, it's possible to push ints, floats and struct instances into the same deque.


## Safe for Concurrent Use
Deque is not safe for concurrent use. However, it's very easy to build a safe for concurrent use version of the deque. Impl7 design document includes an example of how to make impl7 safe for concurrent use using a mutex. Deque can be made safe for concurret use using the same technique. Impl7 design document can be found [here](https://github.com/golang/proposal/blob/master/design/27935-unbounded-queue-package.md).


## Range Support
Just like the current container data structures such as [list](https://github.com/golang/go/tree/master/src/container/list),
[ring](https://github.com/golang/go/tree/master/src/container/ring) and [heap](https://github.com/golang/go/blob/master/src/container/heap/heap.go), deque doesn't support the range keyword for navigation.

However, the API offers two ways to iterate over the deque items. Either use "PopFront"/"PopBack" to retrieve the first current element and the second bool parameter to check for an empty queue.

```go
for v, ok := d.PopFront(); ok; v, ok = d.PopFront() {
    // Do something with v
}
```

Or use "Len" and "PopFront"/"PopBack" to check for an empty deque and retrieve the first current element.
```go
for d.Len() > 0 {
    v, _ := d.PopFront()
    // Do something with v
}
```



## Why
We feel like this world needs improving. Our goal is to change the world, for the better, for everyone.

As software engineers at ef-ds, we feel like the best way we can contribute to a better world is to build amazing systems,
systems that solve real world problems, with unheard performance and efficiency.

We believe in challenging the status-quo. We believe in thinking differently. We believe in progress.

What if we could build queues, stacks, lists, arrays, hash tables, etc that are much faster than the current ones we have? What if we had a dynamic array data structure that offers near constant time deletion (anywhere in the array)? Or that could handle 1 million items data sets using only 1/3 of the memory when compared to all known current implementations? And still runs 2x as fast?

One sofware engineer can't change the world him/herself, but a whole bunch of us can! Please join us improving this world. All the work done here is made 100% transparent and is 100% free. No strings attached. We only require one thing in return: please consider benefiting from it; and if you do so, please let others know about it.


## Support
As the [CloudLogger](https://github.com/cloud-logger/docs) project needed a high performance unbounded queue and, given the fact that Go doesn't provide such queue in its standard library, we built a new queue and proposed it to be added to the standard library.

The initial [proposal](https://github.com/golang/go/issues/27935) was to add [impl7](https://github.com/christianrpetrin/queue-tests/tree/master/queueimpl7/queueimpl7.go) to the standard library.

Given the suggestions to build a deque instead of a FIFO queue as a deque is a much more flexible data structure, coupled with suggestions to build a proper external package, this deque package was built.

We truly believe in the deque and we believe it should have a place in the Go's standard library, so all Go users can benefit from this data structure, and not only the Go insider ones or the lucky ones who found out about the deque by chance or were lucky enough to find it through the search engines.

If you like deque, please help us support it by thumbing up the [proposal](https://github.com/golang/go/issues/27935) and leaving comments.


## Competition
We're extremely interested in improving deque and we're on an endless quest for better efficiency and more performance. Please let us know your suggestions for possible improvements and if you know of other high performance queues not tested here, let us know and we're very glad to benchmark them.


## Releases
We're committed to a CI/CD lifecycle releasing frequent, but only stable, production ready versions with all proper tests in place.

We strive as much as possible to keep backwards compatibility with previous versions, so breaking changes are a no-go.

For a list of changes in each released version, see [CHANGELOG.md](CHANGELOG.md).


## Supported Go Versions
See [supported_go_versions.md](https://github.com/ef-ds/docs/blob/master/supported_go_versions.md).


## License
MIT, see [LICENSE](LICENSE).

"Use, abuse, have fun and contribute back!"


## Contributions
See [CONTRIBUTING.md](CONTRIBUTING.md).


## Roadmap
- Build tool to help find out the combination of firstSliceSize, sliceGrowthFactor, maxFirstSliceSize, maxInternalSliceSize and maxSpareLinks that will yield the best performance
- Build tool to automatically run and calculate the final score of all tests
- Find the fastest open source deques and add them the bench tests
- Improve deque performance and/or efficiency by improving its design and/or implementation
- Build a high performance safe for concurrent use version of deque


## Contact
Suggestions, bugs, new queues to benchmark, issues with the deque, please let us know at ef-ds@outlook.com.
