package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ledgerwatch/log/v3"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
)

var (
	diagnosticsURLFlag = cli.StringFlag{
		Name:  "diagnostics.url",
		Usage: "URL of the diagnostics system provided by the support team, include unique session PIN",
	}
	metricsURLsFlag = cli.StringSliceFlag{
		Name:  "metrics.urls",
		Usage: "Comma separated list of URLs to the metrics endpoints thats are being diagnosed",
	}
	insecureFlag = cli.BoolFlag{
		Name:  "insecure",
		Usage: "Allows communication with diagnostics system using self-signed TLS certificates",
	}
)

var supportCommand = cli.Command{
	Action:    MigrateFlags(connectDiagnostics),
	Name:      "support",
	Usage:     "Connect Erigon instance to a diagnostics system for support",
	ArgsUsage: "--diagnostics.url <URL for the diagnostics system> --metrics.url <http://erigon_host:metrics_port>",
	Flags: []cli.Flag{
		&metricsURLsFlag,
		&diagnosticsURLFlag,
		&insecureFlag,
	},
	Category: "SUPPORT COMMANDS",
	Description: `
The support command connects a running Erigon instances to a diagnostics system specified
by the URL.`,
}

// Conn is client/server symmetric connection.
// It implements the io.Reader/io.Writer/io.Closer to read/write or close the connection to the other side.
// It also has a Send/Recv function to use channels to communicate with the other side.
type Conn struct {
	r  io.Reader
	wc io.WriteCloser

	cancel context.CancelFunc

	wLock sync.Mutex
	rLock sync.Mutex
}

func connectDiagnostics(cliCtx *cli.Context) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigs
		cancel()
	}()

	metricsURLs := cliCtx.StringSlice(metricsURLsFlag.Name)
	metricsURL := metricsURLs[0] // TODO: Generalise

	diagnosticsUrl := cliCtx.String(diagnosticsURLFlag.Name)

	// Create a pool with the server certificate since it is not signed
	// by a known CA
	certPool := x509.NewCertPool()
	srvCert, err := ioutil.ReadFile("diagnostics.crt")
	if err != nil {
		return fmt.Errorf("reading server certificate: %v", err)
	}
	caCert, err := ioutil.ReadFile("CA-cert.pem")
	if err != nil {
		return fmt.Errorf("reading server certificate: %v", err)
	}
	certPool.AppendCertsFromPEM(srvCert)
	certPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration with the certificate of the server
	insecure := cliCtx.Bool(insecureFlag.Name)
	tlsConfig := &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: insecure, //nolint:gosec
	}

	// Perform the requests in a loop (reconnect)
	for {
		if err := tunnel(ctx, tlsConfig, diagnosticsUrl, metricsURL); err != nil {
			return err
		}
		log.Info("Reconnecting in 1 second...")
		timer := time.NewTimer(1 * time.Second)
		select {
		case <-timer.C:
		case <-ctx.Done():
			// Quit immediately if the context was cancelled (by Ctrl-C or TERM signal)
			return nil
		}
	}
}

var successLine = []byte("SUCCESS")

// tunnel operates the tunnel from diagnostics system to the metrics URL for one http/2 request
// needs to be called repeatedly to implement re-connect logic
func tunnel(ctx context.Context, tlsConfig *tls.Config, diagnosticsUrl string, metricsURL string) error {
	diagnosticsClient := &http.Client{Transport: &http2.Transport{TLSClientConfig: tlsConfig}}
	defer diagnosticsClient.CloseIdleConnections()
	metricsClient := &http.Client{}
	defer metricsClient.CloseIdleConnections()
	// Create a request object to send to the server
	reader, writer := io.Pipe()
	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()
	req, err := http.NewRequestWithContext(ctx1, http.MethodPost, diagnosticsUrl, reader)
	if err != nil {
		return err
	}

	// Create a connection
	// Apply given context to the sent request
	resp, err := diagnosticsClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	defer writer.Close()

	// Apply the connection context on the request context
	var metricsBuf bytes.Buffer
	r := bufio.NewReaderSize(resp.Body, 4096)
	line, isPrefix, err := r.ReadLine()
	if isPrefix {
		return fmt.Errorf("request too long")
	}
	if !bytes.Equal(line, successLine) {
		fmt.Errorf("connecting to diagnostics system, first line [%s]", line)
	}
	log.Info("Connected")

	for line, isPrefix, err = r.ReadLine(); err == nil && !isPrefix; line, isPrefix, err = r.ReadLine() {
		fmt.Printf("Got request: %s\n", line)
		metricsBuf.Reset()
		metricsResponse, err := metricsClient.Get(metricsURL + string(line))
		if err != nil {
			fmt.Fprintf(&metricsBuf, "ERROR: Requesting metrics url [%s], query [%s], err: %v", metricsURL, line, err)
		} else {
			// Buffer the metrics response, and relay it back to the diagnostics system, prepending with the size
			if _, err := io.Copy(&metricsBuf, metricsResponse.Body); err != nil {
				metricsBuf.Reset()
				fmt.Fprintf(&metricsBuf, "ERROR: Extracting metrics url [%s], query [%s], err: %v", metricsURL, line, err)
			}
			metricsResponse.Body.Close()
		}
		var sizeBuf [4]byte
		binary.BigEndian.PutUint32(sizeBuf[:], uint32(metricsBuf.Len()))
		if _, err = writer.Write(sizeBuf[:]); err != nil {
			log.Error("Problem relaying metrics prefix len", "url", metricsURL, "query", line, "err", err)
			break
		}
		if _, err = writer.Write(metricsBuf.Bytes()); err != nil {
			log.Error("Problem relaying", "url", metricsURL, "query", line, "err", err)
			break
		}
	}
	if err != nil {
		log.Error("Breaking connection", "err", err)
	}
	if isPrefix {
		log.Error("Request too long, circuit breaker")
	}
	return nil
}
