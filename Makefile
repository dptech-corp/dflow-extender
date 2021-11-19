BIN = bin
ROOT_PKG = argo-job-extender
IMAGE = argo-job-extender:v1.0

.PHONY: all
all: slurm

.PHONY: slurm
slurm:
	go mod vendor && \
	mkdir -p $(BIN) && \
	go build -o $(BIN)/slurm $(ROOT_PKG)/slurm