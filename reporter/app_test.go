package reporter

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless"
	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockGetUsageLimitAPIClient struct {
	mock.Mock
}

func (m *mockGetUsageLimitAPIClient) GetUsageLimit(ctx context.Context, params *redshiftserverless.GetUsageLimitInput, _ ...func(*redshiftserverless.Options)) (*redshiftserverless.GetUsageLimitOutput, error) {
	ret := m.Called(ctx, params)
	output := ret.Get(0)
	if output == nil {
		return nil, ret.Error(1)
	}
	return output.(*redshiftserverless.GetUsageLimitOutput), ret.Error(1)
}

type mockPostMessageAPIClient struct {
	mock.Mock
}

func (m *mockPostMessageAPIClient) PostMessageContext(ctx context.Context, channel string, opts ...slack.MsgOption) (string, string, error) {
	ret := m.Called(ctx, channel, opts)
	return ret.String(0), ret.String(1), ret.Error(2)
}

func TestAppHandler(t *testing.T) {
	mockRedshiftClient := new(mockGetUsageLimitAPIClient)
	mockSlackClient := new(mockPostMessageAPIClient)
	defer func() {
		mockRedshiftClient.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	}()
	mockRedshiftClient.On("GetUsageLimit", mock.Anything, &redshiftserverless.GetUsageLimitInput{
		UsageLimitId: aws.String("00000000-0000-0000-0000-000000000000"),
	}).Return(&redshiftserverless.GetUsageLimitOutput{
		UsageLimit: &types.UsageLimit{
			Amount:        aws.Int64(100),
			BreachAction:  types.UsageLimitBreachActionEmitMetric,
			Period:        types.UsageLimitPeriodDaily,
			ResourceArn:   aws.String("arn:aws:redshift-serverless:ap-northeast-1:123456789912:workgroup/00000000-0000-0000-0000-000000000001"),
			UsageLimitArn: aws.String("arn:aws:redshift-serverless:ap-northeast-1:12345789012:usagelimit/00000000-0000-0000-0000-000000000000"),
			UsageLimitId:  aws.String("00000000-0000-0000-0000-000000000000"),
			UsageType:     types.UsageLimitUsageTypeServerlessCompute,
		},
	}, nil).Times(1)
	mockSlackClient.On("PostMessageContext", mock.Anything, "channel", mock.Anything).Return("channel", "123456789", nil).Times(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bs, err := os.ReadFile("testdata/redshift_serverless_limit_usage_payload.json")
	require.NoError(t, err)
	app := NewAppWithClient(mockRedshiftClient, mockSlackClient, "channel")
	_, err = app.Handler(ctx, bs)
	require.NoError(t, err)
}
