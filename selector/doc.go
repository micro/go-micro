/*
Package selector is a way to load balance service nodes.

It algorithmically filter and return nodes required by the client or any other system.
Selector's implemented by Micro build on the registry but it's of optional use. One could
provide a static Selector that has a fixed pool.
*/
package selector
