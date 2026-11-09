// Package app is the command line interface of hexlet-go-crawler.
package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"code/crawler"
)

const version = "1.0.0"

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.UintFlag{Name: "depth", Value: 10, Usage: "crawl depth"},
		&cli.UintFlag{Name: "retries", Value: 1, Usage: "number of retries for failed requests"},
		&cli.DurationFlag{Name: "delay", Value: 0, Usage: "delay between requests (example: 200ms, 1s)"},
		&cli.DurationFlag{Name: "timeout", Value: 15 * time.Second, Usage: "per-request timeout"},
		&cli.UintFlag{Name: "rps", Value: 0, Usage: "limit requests per second (overrides delay)"},
		&cli.StringFlag{Name: "user-agent", Usage: "custom user agent"},
		&cli.UintFlag{Name: "workers", Value: 4, Usage: "number of concurrent workers"},
		&cli.BoolFlag{Name: "indent-json", Value: true, Usage: "indent the JSON report"},
	}
}

const maxRPS = uint(time.Second / time.Nanosecond)

func resolveDelay(rps uint, delay time.Duration) (time.Duration, error) {
	if delay < 0 {
		return 0, fmt.Errorf("%w: %s", crawler.ErrInvalidDelay, delay)
	}

	if rps == 0 {
		return delay, nil
	}

	if rps > maxRPS {
		return 0, fmt.Errorf("invalid rps: %d", rps)
	}

	return time.Second / time.Duration(rps), nil
}

func run(ctx context.Context, cmd *cli.Command) error {
	urlArg := cmd.Args().First()
	if urlArg == "" {
		return cli.ShowRootCommandHelp(cmd)
	}

	if cmd.NArg() > 1 {
		args := strings.Join(cmd.Args().Slice(), " ")

		return fmt.Errorf("expected 1 url path, got '%s'", args)
	}

	delay, err := resolveDelay(cmd.Uint("rps"), cmd.Duration("delay"))
	if err != nil {
		return err
	}

	data, err := crawler.Analyze(ctx, crawlerOptions(cmd, delay))
	if err != nil {
		return fmt.Errorf("analyze error: %w", err)
	}

	_, err = fmt.Fprintln(cmd.Root().Writer, string(data))

	return err
}

func crawlerOptions(cmd *cli.Command, delay time.Duration) crawler.Options {
	return crawler.Options{
		URL:         cmd.Args().First(),
		Depth:       cmd.Uint("depth"),
		Retries:     cmd.Uint("retries"),
		Delay:       delay,
		Timeout:     cmd.Duration("timeout"),
		UserAgent:   cmd.String("user-agent"),
		Concurrency: cmd.Uint("workers"),
		HTTPClient:  &http.Client{},
		IndentJSON:  cmd.Bool("indent-json"),
	}
}

func newCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:        "hexlet-go-crawler",
		Description: "analyze a website structure",
		Usage:       "analyze a website structure",
		ArgsUsage:   "<url>",
		Writer:      stdout,
		ErrWriter:   stderr,
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "print version",
				Action: func(_ context.Context, cmd *cli.Command) error {
					_, err := fmt.Fprintln(cmd.Root().Writer, version)

					return err
				},
			},
		},
		Flags:  flags(),
		Action: run,
	}
}

// Run executes the CLI with args and returns the process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if err := newCommand(stdout, stderr).Run(ctx, args); err != nil {
		_, _ = fmt.Fprintln(stderr, err)

		return 1
	}

	return 0
}
