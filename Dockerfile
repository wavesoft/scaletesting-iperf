FROM golang:1.14-alpine
WORKDIR /build
COPY . /build
RUN go build -o iperf-runner

FROM alpine:3
RUN wget -O /usr/local/bin/iperf3 https://github.com/userdocs/iperf3-static/raw/master/bin/iperf3 && \
    chmod +x /usr/local/bin/iperf3
COPY --from=0 /build/iperf-runner /usr/local/bin/iperf-runner
ENTRYPOINT [ "/usr/local/bin/iperf-runner" ]
