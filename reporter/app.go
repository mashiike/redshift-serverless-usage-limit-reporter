package reporter

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless"
	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	"github.com/slack-go/slack"
)

type GetUsageLimitAPIClient interface {
	GetUsageLimit(ctx context.Context, params *redshiftserverless.GetUsageLimitInput, optFns ...func(*redshiftserverless.Options)) (*redshiftserverless.GetUsageLimitOutput, error)
}

type PostMessageAPIClient interface {
	PostMessageContext(ctx context.Context, channel string, opts ...slack.MsgOption) (string, string, error)
}

type AppOption func(*App)

func WithLogger(logger *slog.Logger) AppOption {
	return func(app *App) {
		app.logger = logger
	}
}

func WithReportTemplate(tmpl *template.Template) AppOption {
	return func(app *App) {
		app.tmpl = tmpl
	}
}

func WithRetryCount(count int) AppOption {
	return func(app *App) {
		app.retryCount = count
	}
}

type App struct {
	retryCount     int
	tmpl           *template.Template
	logger         *slog.Logger
	redshiftClient GetUsageLimitAPIClient
	slackClient    PostMessageAPIClient
	slackChannel   string
}

func NewApp(ctx context.Context, slackBotToken string, slackChannel string, opts ...AppOption) (*App, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}
	redshiftClient := redshiftserverless.NewFromConfig(awsCfg)
	slackClient := slack.New(slackBotToken)
	resp, err := slackClient.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate slack client: %w", err)
	}
	app := NewAppWithClient(redshiftClient, slackClient, slackChannel, opts...)
	app.logger.Info("slack client authenticated", "bot_id", resp.BotID, "team_id", resp.TeamID, "user_id", resp.UserID)
	return app, nil
}

//go:embed default_report.tmpl.json
var defaultReportTemplateString string
var defaultReportTemplate *template.Template

func init() {
	defaultReportTemplate = template.Must(template.New("default_report").Parse(defaultReportTemplateString))
}

func NewAppWithClient(redshiftClient GetUsageLimitAPIClient, slackClient PostMessageAPIClient, slackChannel string, opts ...AppOption) *App {
	app := &App{
		tmpl:           defaultReportTemplate,
		logger:         slog.Default().With("app", "redshift-serverless-usage-limit-reporter"),
		redshiftClient: redshiftClient,
		slackChannel:   slackChannel,
		slackClient:    slackClient,
		retryCount:     3,
	}
	for _, opt := range opts {
		opt(app)
	}
	return app
}

func (app *App) Handler(ctx context.Context, raw json.RawMessage) (interface{}, error) {
	var event events.SNSEvent
	if err := json.Unmarshal(raw, &event); err == nil && len(event.Records) > 0 {
		var errs []error
		for _, record := range event.Records {
			if err := app.handler(ctx, app.logger.With("sns_message_id", record.SNS.MessageID), []byte(record.SNS.Message)); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}
		return nil, nil
	}
	if err := app.handler(ctx, app.logger, raw); err != nil {
		return nil, err
	}
	return nil, nil
}

func (app *App) handler(ctx context.Context, logger *slog.Logger, bs []byte) error {
	var message CloudWatchAlermMessage
	if err := json.Unmarshal(bs, &message); err != nil {
		logger.Error("failed to unmarshal message", "error", err)
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	logger = logger.With("alarm_name", message.AlarmName, "alarm_description", message.AlarmDescription)
	if !message.IsRedshiftServerlessLimitUsage() {
		logger.Debug("skip message", "reason", "not redshift serverless limit usage")
		return nil
	}
	if err := app.SendReport(ctx, message); err != nil {
		logger.Error("failed to send report", "error", err)
		return fmt.Errorf("failed to send report: %w", err)
	}
	return nil
}

func (app *App) newTemplateData(message CloudWatchAlermMessage, output *redshiftserverless.GetUsageLimitOutput) map[string]interface{} {
	var usageLimitAmountUnit string
	switch output.UsageLimit.UsageType {
	case types.UsageLimitUsageTypeServerlessCompute:
		usageLimitAmountUnit = "RPU hours"
	case types.UsageLimitUsageTypeCrossRegionDatasharing:
		usageLimitAmountUnit = "TB"
	default:
		usageLimitAmountUnit = ""
	}
	return map[string]interface{}{
		"Message":              message,
		"UsageLimit":           output.UsageLimit,
		"WorkGroupName":        message.WorkgroupName(),
		"UsageLimitAmountUnit": usageLimitAmountUnit,
		"SenderInfo":           app.senderInfo(message),
	}
}

func (app *App) senderInfo(message CloudWatchAlermMessage) string {
	var builder strings.Builder
	builder.WriteString("this report send by ")
	if lambdacontext.FunctionName != "" {
		fmt.Fprintf(&builder, "lambda function %s (version: %s)", lambdacontext.FunctionName, lambdacontext.FunctionVersion)
	} else {
		builder.WriteString("redshift-serverless-usage-limit-reporter")
	}
	if message.AlarmArn != "" {
		builder.WriteString("\\n trigered by ")
		builder.WriteString(message.AlarmArn)
	}
	return builder.String()
}

type slackMessage struct {
	slack.PostMessageParameters
	Text        string             `json:"text,omitempty"`
	Blocks      slack.Blocks       `json:"blocks,omitempty"`
	Attachments []slack.Attachment `json:"attachments,omitempty"`
}

func (app *App) newSlackMessageBlocks(data map[string]interface{}) ([]slack.MsgOption, error) {
	var buf bytes.Buffer
	if err := app.tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	var msg slackMessage
	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		return nil, err
	}
	opts := []slack.MsgOption{
		slack.MsgOptionPostMessageParameters(msg.PostMessageParameters),
	}
	if msg.Text != "" {
		opts = append(opts, slack.MsgOptionText(msg.Text, false))
	}
	if msg.Blocks.BlockSet != nil {
		opts = append(opts, slack.MsgOptionBlocks(msg.Blocks.BlockSet...))
	}
	if msg.Attachments != nil {
		opts = append(opts, slack.MsgOptionAttachments(msg.Attachments...))
	}
	return opts, nil
}

func (app *App) SendReport(ctx context.Context, message CloudWatchAlermMessage) error {
	output, err := app.redshiftClient.GetUsageLimit(ctx, &redshiftserverless.GetUsageLimitInput{
		UsageLimitId: aws.String(message.UsageLimitID()),
	})
	if err != nil {
		return fmt.Errorf("failed to get usage limit: %w", err)
	}
	data := app.newTemplateData(message, output)
	opts, err := app.newSlackMessageBlocks(data)
	if err != nil {
		return fmt.Errorf("failed to create slack message blocks: %w", err)
	}
	var count int
	var ts string
	for count <= app.retryCount {
		_, ts, err = app.slackClient.PostMessageContext(
			ctx, app.slackChannel, opts...,
		)
		if err == nil {
			break
		}
		var rateLimitedErr *slack.RateLimitedError
		if errors.As(err, &rateLimitedErr) {
			app.logger.Warn("slack rate limit exceeded", "retry_after", rateLimitedErr.RetryAfter)
			count++
			continue
		}
		app.logger.Error("failed to send slack message", "error", err)
		return err
	}
	app.logger.Info("slack message sent", "channel", app.slackChannel, "ts", ts)
	return err
}
