PROJECT_ROOT = D:/Gyat

CC=go
GO_SRC_DIR=$(PROJECT_ROOT)/cmd/gyat
GO_BIN_LOCATION=gyat.exe

_GO_FILES = main.go cmd.go util.go
GO_FILES = $(patsubst %,$(GO_SRC_DIR)/%,$(_GO_FILES))

gyat.exe:
	$(CC) build -o $(GO_BIN_LOCATION) $(GO_FILES)

.PHONY: clean setup clean-bin

TEST_FILE=test.txt

# add more setups
setup:
	./gyat init \
	&& touch $(TEST_FILE) \
	&& echo "Hello, Gyat" > $(TEST_FILE) \
	&& ./gyat hash-object -w $(TEST_FILE)

clean:
	rm -rf $(GO_BIN_LOCATION) .gyat *.txt