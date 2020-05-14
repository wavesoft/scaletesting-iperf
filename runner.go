package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/quipo/statsd"
	log "github.com/sirupsen/logrus"
)

const (
	TYPE_GAUGE     = 1
	TYPE_INCREMENT = 2
	TYPE_DECREMENT = 3
)

type PerfMetric struct {
	Name  string
	Value int64
	Type  int
}

func iperfExecute(args []string, stdout chan string, stderr chan string, exit chan int) {
	log.Infof("Starting iperf %s", strings.Join(args, " "))
	proc := exec.Command("/usr/local/bin/iperf3", args...)

	stdoutPipe, err := proc.StdoutPipe()
	if err != nil {
		log.Fatalf("Unable to open StdOut Pipe: %s", err.Error())
	}
	stderrErr, err := proc.StderrPipe()
	if err != nil {
		log.Fatalf("Unable to open StdErr Pipe: %s", err.Error())
	}
	err = proc.Start()
	if err != nil {
		proc = nil
		log.Fatalf("Could not start iperf: %s", err.Error())
	}

	// Async readers of the Stdout/Err
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			stdout <- scanner.Text()
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderrErr)
		for scanner.Scan() {
			stderr <- scanner.Text()
		}
	}()

	// Wait for the process to exit
	err = proc.Wait()
	if xerr, ok := err.(*exec.ExitError); ok {
		log.Errorf("Process exited with code %d", xerr.ExitCode())
		exit <- xerr.ExitCode()
	} else if err != nil {
		log.Errorf("Process exited with error: %s", err.Error())
		exit <- -1
	} else {
		log.Warnf("Process exited cleanly")
		exit <- 0
	}
}

func byteExprToInt(num string, scale string, scaleBase float64) int64 {
	value, err := strconv.ParseFloat(num, 64)
	if err != nil {
		log.Fatalf("Could not parse value: '%s' as a float-point number", num)
	}

	switch strings.ToLower(scale) {
	case "k":
		value *= scaleBase
	case "m":
		value *= scaleBase * scaleBase
	case "g":
		value *= scaleBase * scaleBase * scaleBase
	case "t":
		value *= scaleBase * scaleBase * scaleBase * scaleBase
	}

	return int64(value)
}

func iperfParser(stdout chan string, stderr chan string, exit chan int, ch chan *PerfMetric) {
	rxBitrateLine := regexp.MustCompile(`\s([0-9\.]+)\s+([KMGT])Bytes\s+([0-9\.]+) ([KMGT])Bytes\/sec`)

	for {
		select {
		case line := <-stdout:
			log.Info(line)

			// Extract bit rate information and push them as metrics
			found := rxBitrateLine.FindStringSubmatch(line)
			if len(found) > 0 {
				txBytes := byteExprToInt(found[1], found[2], 1024)
				ch <- &PerfMetric{
					Name:  "bytes",
					Value: int64(txBytes),
					Type:  TYPE_INCREMENT,
				}

				txBitrate := byteExprToInt(found[3], found[4], 1000)
				ch <- &PerfMetric{
					Name:  "bitrate",
					Value: int64(txBitrate),
					Type:  TYPE_GAUGE,
				}
			}

		case line := <-stderr:
			log.Warn(line)
		case code := <-exit:
			ch <- &PerfMetric{
				Name:  "status",
				Value: int64(code),
				Type:  TYPE_GAUGE,
			}
			return
		}
	}
}

func statsdForwarder(host string, port int, prefix string, ch chan *PerfMetric) {
	log.Infof("Forwarding stats on endpoint %s:%d", host, port)
	statsdclient := statsd.NewStatsdClient(fmt.Sprintf("%s:%d", host, port), prefix)
	err := statsdclient.CreateSocket()
	if nil != err {
		log.Fatalf("Could not start statsd forwarder: %s", err.Error())
	}

	interval := time.Second * 2 // aggregate stats and flush every 2 seconds
	stats := statsd.NewStatsdBuffer(interval, statsdclient)
	defer stats.Close()

	for {
		select {
		case metric := <-ch:

			switch metric.Type {
			case TYPE_GAUGE:
				stats.Gauge(metric.Name, metric.Value)
				log.Debugf("metric %s%s = %d", prefix, metric.Name, metric.Value)

			case TYPE_INCREMENT:
				stats.Incr(metric.Name, metric.Value)
				log.Debugf("metric %s%s += %d", prefix, metric.Name, metric.Value)

			case TYPE_DECREMENT:
				stats.Decr(metric.Name, metric.Value)
				log.Debugf("metric %s%s -= %d", prefix, metric.Name, metric.Value)
			}
		}
	}
}

func startIPerf(args []string, ch chan *PerfMetric) {
	stdout := make(chan string)
	stderr := make(chan string)
	exitCode := make(chan int)

	go iperfParser(stdout, stderr, exitCode, ch)
	iperfExecute(args, stdout, stderr, exitCode)
}

func main() {

	// Get configuration options
	statsdHost := os.Getenv("STATSD_UDP_HOST")
	if statsdHost == "" {
		log.Fatalf("Missing environment variable STATSD_UDP_HOST")
	}

	statsdPortStr := os.Getenv("STATSD_UDP_PORT")
	if statsdPortStr == "" {
		log.Fatalf("Missing environment variable STATSD_UDP_PORT")
	}
	statsdPort, err := strconv.Atoi(statsdPortStr)
	if err != nil {
		log.Fatalf("Expecting STATSD_UDP_PORT to be a number")
	}

	statsdPrefix := os.Getenv("STATSD_PREFIX")

	iperfMode := strings.ToLower(os.Getenv("IPERF_SIDE"))
	if iperfMode == "" {
		log.Fatalf("Missing environment variable IPERF_SIDE")
	}

	iperfHost := strings.ToLower(os.Getenv("IPERF_HOST"))

	iperfPortStr := os.Getenv("IPERF_PORT")
	if iperfPortStr == "" {
		iperfPortStr = "5201"
	}
	iperfPort, err := strconv.Atoi(iperfPortStr)
	if err != nil {
		log.Fatalf("Expecting IPERF_PORT to be a number")
	}

	iperfParallelStr := os.Getenv("IPERF_PARALLEL")
	if iperfParallelStr == "" {
		iperfParallelStr = "1"
	}
	iperfParallel, err := strconv.Atoi(iperfParallelStr)
	if err != nil {
		log.Fatalf("Expecting IPERF_PARALLEL to be a number")
	}

	restartSecondsStr := os.Getenv("RESTART_SCONDS")
	if restartSecondsStr == "" {
		restartSecondsStr = "10"
	}
	restartSeconds, err := strconv.Atoi(restartSecondsStr)
	if err != nil {
		log.Fatalf("Expecting RESTART_SCONDS to be a number")
	}

	// Prepare iperf configuration
	args := []string{
		"-f", "K", // Report values in kBit
		"--forceflush", // Always flush intervals
		"-p", fmt.Sprintf("%d", iperfPort),
	}

	if iperfMode == "server" {
		if iperfHost == "" {
			iperfHost = "0.0.0.0"
		}
		if statsdPrefix == "" {
			statsdPrefix = "perf.server."
		}

		args = append(args,
			"-s", iperfHost,
		)

	} else if iperfMode == "client" {
		if iperfHost == "" {
			log.Fatalf("Missing environment variable IPERF_HOST")
		}
		if statsdPrefix == "" {
			statsdPrefix = "perf.client."
		}

		args = append(args,
			"-c", iperfHost,
			"-P", fmt.Sprintf("%d", iperfParallel),
		)

		// Cap bit rate if configured
		iperfBitrate := os.Getenv("IPERF_BITRATE")
		if iperfBitrate != "" {
			args = append(args,
				"-b", iperfBitrate,
			)
		}

		// Use UDP if configured
		iperfUdp := strings.ToLower(os.Getenv("IPERF_UDP"))
		if iperfUdp == "1" || iperfUdp == "yes" || iperfUdp == "true" {
			args = append(args,
				"-u",
			)
		}

	} else {
		log.Fatalf("Expecting IPERF_SIDE to be 'server' or 'client'")
	}

	// Append additional parameters
	extraArgs := os.Getenv("IPERF_EXTRA_ARGS")
	if extraArgs != "" {
		args = append(args, strings.Fields(extraArgs)...)
	}

	// Start the statsd forwarder
	metricsChan := make(chan *PerfMetric)
	go statsdForwarder(statsdHost, statsdPort, statsdPrefix, metricsChan)

	// Start the runner
	for {
		metricsChan <- &PerfMetric{
			Name:  "status",
			Value: int64(-1),
			Type:  TYPE_GAUGE,
		}
		metricsChan <- &PerfMetric{
			Name:  "running",
			Value: int64(1),
			Type:  TYPE_INCREMENT,
		}
		startIPerf(args, metricsChan)
		metricsChan <- &PerfMetric{
			Name:  "running",
			Value: int64(1),
			Type:  TYPE_DECREMENT,
		}

		log.Infof("Going to re-start in %d seconds...", restartSeconds)
		time.Sleep(time.Second * time.Duration(restartSeconds))
	}
}
