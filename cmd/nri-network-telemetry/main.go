package main

import "fmt"

var (
	// Version is your app version (updated by Makefile, don't forget to TAG YOUR RELEASE)
	Version = "Unknown"
)

func main() {
	fmt.Printf("Example App version: %s\n", Version)
}
