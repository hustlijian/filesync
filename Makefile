
all: *.go */*.go
	go build client.go
	go build server.go

clean:
	rm -vf client server
	rm -vf *.exe
