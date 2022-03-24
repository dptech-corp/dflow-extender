FROM golang:latest

RUN apt update
RUN apt install -y sshpass
WORKDIR /data/argo-job-extender
COPY ./ ./
RUN make
