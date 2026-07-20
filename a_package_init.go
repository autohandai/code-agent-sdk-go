package autohand

import "time"

// The benchmark probe reads this duration from a fresh child after package
// initialization. The leading filename keeps the start marker before the
// package's other variable initializers in the Go toolchain's lexical file order.
var packageInitializationStarted = time.Now()
var packageInitializationElapsed time.Duration

func init() {
	packageInitializationElapsed = time.Since(packageInitializationStarted)
}
