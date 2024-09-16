# Mutual Exclusion 
Implementation of distributed systems Ricart-Agrawala algorithm.
Allows processes to use a shared resource avoiding race coditions.


## How to build
```bash
go build Process.go
go build SharedResource.go
```

## How to Run
```bash
./Process 1 :10002 :10003 :10004
./Process 2 :10002 :10003 :10004
./Process 3 :10002 :10003 :10004
./SharedResource
```
