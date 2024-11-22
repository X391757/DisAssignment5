# How to run the program

Open three terminals and enter the following command respectively.

```go
go run main.go -port=8080
```

```go
go run main.go -port=8081
```

```go
cd coordinator
go run coordinator.go
```

For testing, you can enter 1 id1 500 to indicate that the user with id1 bid 500, and enter 2 to query the current auction results.