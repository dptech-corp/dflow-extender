BIN = bin
ROOT_PKG = dflow-extender
IMAGE = dflow-extender:v1.0

.PHONY: all
all: slurm

.PHONY: slurm
slurm:
	go mod vendor && \
	mkdir -p $(BIN) && \
	go build -o $(BIN)/slurm $(ROOT_PKG)/slurm
