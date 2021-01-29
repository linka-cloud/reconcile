
.PHONY:
proto:
	@protoc -I. --go_out=. storage/proto/record.proto
