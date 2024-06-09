CC=go
GO_MAIN=cmd/gyat/main.go
GO_BIN_LOCATION=gyat.exe

gyat.exe:
	$(CC) build -o $(GO_BIN_LOCATION) $(GO_MAIN)

.PHONY: clean

clean:
	rm -rf $(GO_BIN_LOCATION) .gyat