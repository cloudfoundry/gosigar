package main

func main() {
	for {
		fib(50) //nolint:staticcheck
	}
}

func fib(n int) int {
	if n <= 1 {
		return 1
	}
	return fib(n-1) + fib(n-2) //nolint:staticcheck
}
