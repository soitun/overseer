// A tiny helper binary used by proc_master_panic_integration_test.go.
// It panics on startup so the test can assert OnPanic fires.
package main

func main() {
	panic("synthetic panic from panicker helper")
}
