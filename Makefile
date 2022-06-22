BIN = bin

.PHONY: all
all: slurm

.PHONY: slurm
slurm:
	go mod vendor && \
	mkdir -p $(BIN) && \
	go build -o $(BIN)/slurm ./cmd/slurm/slurm.go
