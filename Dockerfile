FROM golang:latest

WORKDIR /data/argo-job-extender
COPY ./ ./
RUN make
