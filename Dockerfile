FROM golang:latest

RUN apt update
RUN apt install -y sshpass
WORKDIR /data/dflow-extender
COPY ./ ./
RUN make
