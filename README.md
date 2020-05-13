# scaletesting-iperf
> IPerf3 executor for running scale-tests

## Usage

You can run pairs of the `iperf` container in you cluster for creating constant TCP or UDP streams of test data. Each instance can feed it's current observations to a StatsD server.

```sh
docker run -it --rm \
    -e STATSD_UDP_HOST=statsd.server.local \
    -e STATSD_UDP_PORT=8125 \
    -e IPERF_SIDE=server \
    icharalampidis/projects:scaletesting-iperf
```

## Configuration

The following environment variables can (or must) be provided:

<table>
    <tr>
        <th>Variable</th>
        <th>Default</th>
        <th>Description</th>
    </tr>
    <tr>
        <th><code>STATSD_UDP_HOST</code></th>
        <td>(Required)</td>
        <td>The hostname of the STATSD server where to push the metrics.</td>
    </tr>
    <tr>
        <th><code>STATSD_UDP_PORT</code></th>
        <td>(Required)</td>
        <td>The port of the STATSD server where to push the metrics.</td>
    </tr>
    <tr>
        <th rowspan="2"><code>STATSD_PREFIX</code></th>
        <td><code>perf.server.</code></td>
        <td rowspan="2">The prefix for all metrics emitted by the runner.</td>
    </tr>
    <tr>
        <td><code>perf.client.</code></td>
    </tr>
    <tr>
        <th><code>IPERF_SIDE</code></th>
        <td>(Required)</td>
        <td>Specify either <code>server</code> or <code>client</code>.</td>
    </tr>
    <tr>
        <th><code>IPERF_HOST</code></th>
        <td>0.0.0.0</td>
        <td>The host to either listen to or connect to.</td>
    </tr>
    <tr>
        <th><code>IPERF_PORT</code></th>
        <td>5201</td>
        <td>The TCP/UDP port to use for sending traffic through.</td>
    </tr>
    <tr>
        <th><code>IPERF_PARALLEL</code></th>
        <td>1</td>
        <td>How many parallel streams to establish.</td>
    </tr>
    <tr>
        <th><code>IPERF_BITRATE</code></th>
        <td>0</td>
        <td>Target bitrate in bits/sec (0 for unlimited). Default 1 Mbit/sec for UDP, unlimited for TCP, optional slash and packet count for burst mode.</td>
    </tr>
    <tr>
        <th><code>IPERF_UDP</code></th>
        <td>no</td>
        <td>Set to <code>yes</code> to use UDP instead of TCP.</td>
    </tr>
    <tr>
        <th><code>IPERF_EXTRA_ARGS</code></th>
        <td></td>
        <td>Additional arguments to pass to the iperf3 binary.</td>
    </tr>
    <tr>
        <th><code>RESTART_SCONDS</code></th>
        <td>10</td>
        <td>How many seconds to wait before re-starting the client (or server) after it exists.</td>
    </tr>
</table>

## Metrics

The service is pushing the following metrics to the STATSD endpoint:

<table>
    <tr>
        <th>Metric</th>
        <th>Type</th>
        <th>Units</th>
        <th>Description</th>
    </tr>
    <tr>
        <th><code>start</code></th>
        <td>Counter</td>
        <td>-</td>
        <td>Counts how many times the process has started.</td>
    </tr>
    <tr>
        <th><code>status</code></th>
        <td>Gauge</td>
        <td>-</td>
        <td>Indicates the last exit code.</td>
    </tr>
    <tr>
        <th><code>bytes</code></th>
        <td>Counter</td>
        <td>Bytes</td>
        <td>Counts how many bytes have been sent/received during the last session.</td>
    </tr>
    <tr>
        <th><code>bitrate</code></th>
        <td>Gauge</td>
        <td>Bits/sec</td>
        <td>Indicates the current throughput.</td>
    </tr>
</table>
