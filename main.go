package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/floj/kostal2influx/kostal"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "kostal2influx",
		Usage: "Published Kostal Plenticore MODBUS metrics to Influx-DB",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "influx-url",
				Usage:    "Influx-DB url, e.g. http://localhost:8080/api/v2/write?org=YOUR_ORG&bucket=YOUR_BUCKET",
				EnvVars:  []string{"INFLUX_URL"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "influx-token",
				Usage:    "Influx-DB access token",
				EnvVars:  []string{"INFLUX_TOKEN"},
				Required: true,
			},
			&cli.DurationFlag{
				Name:    "influx-timeout",
				Usage:   "Influx-DB request timeout",
				EnvVars: []string{"INFLUX_TIMEOUT"},
				Value:   time.Second * 5,
			},
			&cli.StringFlag{
				Name:     "kostal-addr",
				Usage:    "The Kostal MODBUS-TCP address, e.g. 192.165.1.30:1502",
				EnvVars:  []string{"KOSTAL_ADDR"},
				Required: true,
			},
			&cli.DurationFlag{
				Name:    "interval",
				Usage:   "Frequency how often the metrics are published",
				EnvVars: []string{"INTERVAL"},
				Value:   time.Second * 10,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Usage:   "Display more information",
				EnvVars: []string{"VERBOSE"},
				Aliases: []string{"v"},
				Value:   false,
			},
		},
		Action: run,
	}

	err := app.Run(os.Args)
	log.Println("done")
	if err != nil {
		log.Fatal(err)
	}

}

func run(ctx *cli.Context) error {
	influxURL := ctx.String("influx-url")
	influxToken := ctx.String("influx-token")
	verbose := ctx.Bool("verbose")

	kc, err := kostal.NewClient(ctx.String("kostal-addr"), verbose)
	if err != nil {
		return err
	}
	defer kc.Close()

	log.Printf("connected to Kostal %s", kc.SerialNumber())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	httpClient := &http.Client{
		Timeout: ctx.Duration("influx-timeout"),
	}

	if err := readAndPublish(kc, httpClient, influxURL, influxToken, verbose); err != nil {
		return err
	}

	log.Printf("publishing metrics every %v", ctx.Duration("interval"))
	ticker := time.NewTicker(ctx.Duration("interval"))
	defer ticker.Stop()

	for {
		select {
		case s := <-sigs:
			log.Printf("%s received, stopping\n", s)
			return nil
		case t := <-ticker.C:
			log.Printf("reading registers at %s\n", t)
			err := readAndPublish(kc, httpClient, influxURL, influxToken, verbose)
			if err != nil {
				return err
			}
		}
	}
}

func readAndPublish(kc *kostal.Client, httpClient *http.Client, influxURL, influxToken string, verbose bool) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "pv,device=%s,vendor=kostal ", kc.SerialNumber())

	regs, err := kc.ReadAll(func(r kostal.Register) bool {
		return r.Include
	})
	if err != nil {
		return err
	}

	for i, r := range regs {
		if verbose {
			log.Printf("%d | %s | %v | %s\n", r.Register.Addr, r.Register.Description, r.Value, r.Register.InfluxField)
		}
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, "%s=%v", r.Register.InfluxField, r.Value)
	}

	req, err := http.NewRequest("POST", influxURL, &buf)
	if err != nil {
		return fmt.Errorf("could not create http request: %w", err)
	}
	req.Header.Add("Authorization", "Token "+influxToken)
	req.Header.Add("Content-Type", "text/plain; charset=utf-8")
	req.Header.Add("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send metrics to influxdb: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected influx to return %d, but got: %d - %s", http.StatusNoContent, resp.StatusCode, body)
	}

	return nil
}
