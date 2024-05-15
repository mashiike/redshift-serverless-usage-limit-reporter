package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"text/template"

	"github.com/fujiwara/lamblocal"
	"github.com/handlename/ssmwrap/v2"
	"github.com/ken39arg/go-flagx"
	"github.com/mashiike/redshift-serverless-usage-limit-reporter/reporter"
)

var Version string = "0.0.0"

func main() {
	setupLogger("error")
	if err := _main(); err != nil {
		slog.Error(err.Error())
	}
}

func setupLogger(minLevel string) {
	var level slog.Level
	switch minLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		},
		))
	slog.SetDefault(logger)
}

func _main() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	exportRules := make([]ssmwrap.ExportRule, 0)
	if path := os.Getenv("SSMWRAP_PATH"); path != "" {
		exportRules = append(exportRules, ssmwrap.ExportRule{
			Path: path,
		})
	}
	if prefix := os.Getenv("SSMWRAP_PREFIX"); prefix != "" {
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		exportRules = append(exportRules, ssmwrap.ExportRule{
			Path: prefix + "*",
		})
	}
	if len(exportRules) > 0 {
		if err := ssmwrap.Export(ctx, exportRules, ssmwrap.ExportOptions{Retries: 3}); err != nil {
			return fmt.Errorf("failed to ssmwrap.Export: %w", err)
		}
	}
	var (
		slackBotToken       string
		slackChannel        string
		reportTemplateFile  string
		sendMessageMaxRetry int
		logLevel            string
		showVersion         bool
	)
	flag.StringVar(&slackBotToken, "slack-bot-token", "", "Slack bot token")
	flag.StringVar(&slackChannel, "slack-channel", "", "Slack channel")
	flag.StringVar(&reportTemplateFile, "report-template-file", "", "Report template file")
	flag.StringVar(&logLevel, "log-level", "info", "Log level")
	flag.IntVar(&sendMessageMaxRetry, "send-message-max-retry", 3, "Max retry count to send message")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.VisitAll(flagx.EnvToFlag)
	flag.Parse()
	setupLogger(logLevel)

	slog.Info("redshift-serverless-usage-limit-reporter", "version", Version)
	if showVersion {
		slog.Info("go runtime version", "version", runtime.Version())
		return nil
	}
	appOpts := []reporter.AppOption{}
	if slackBotToken == "" {
		return errors.New("slack-bot-token is required")
	}
	if slackChannel == "" {
		return errors.New("slack-channel is required")
	}
	if reportTemplateFile != "" {
		tmpl, err := template.ParseFiles(reportTemplateFile)
		if err != nil {
			return fmt.Errorf("failed to parse report template file: %w", err)
		}
		appOpts = append(appOpts, reporter.WithReportTemplate(tmpl))
	}
	appOpts = append(appOpts, reporter.WithRetryCount(sendMessageMaxRetry))
	app, err := reporter.NewApp(ctx, slackBotToken, slackChannel, appOpts...)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}
	return lamblocal.RunWithError(ctx, app.Handler)
}
