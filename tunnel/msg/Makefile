message.pb.go: message.proto
	protoc --go_out=. --go_opt=paths=source_relative $<

.PHONY: clean
clean:
	rm -f *.pb.go
