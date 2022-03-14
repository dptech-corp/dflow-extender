FROM golang:latest

WORKDIR /data/argo-job-extender
COPY ./ ./
RUN make
RUN apt update
RUN apt install -y sshpass
