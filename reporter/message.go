package reporter

import "strings"

type CloudWatchAlermMessage struct {
	AlarmName                          string                        `json:"AlarmName"`
	AlarmDescription                   *string                       `json:"AlarmDescription"`
	AWSAccountID                       string                        `json:"AWSAccountId"`
	AlarmConfigurationUpdatedTimestamp string                        `json:"AlarmConfigurationUpdatedTimestamp"`
	NewStateValue                      string                        `json:"NewStateValue"`
	NewStateReason                     string                        `json:"NewStateReason"`
	OldStateValue                      string                        `json:"OldStateValue"`
	StateChangeTime                    string                        `json:"StateChangeTime"`
	AWSRegion                          string                        `json:"Region"`
	AlarmArn                           string                        `json:"AlarmArn"`
	Trigger                            CloudWatchAlermMessageTrigger `json:"Trigger"`
}

type CloudWatchAlermMessageTrigger struct {
	MetricName                       string                                   `json:"MetricName"`
	Namespace                        string                                   `json:"Namespace"`
	StatisticType                    string                                   `json:"StatisticType"`
	Statistic                        string                                   `json:"Statistic"`
	Unit                             any                                      `json:"Unit"`
	Dimensions                       []CloudWatchAlermMessageTriggerDimension `json:"Dimensions"`
	Period                           int                                      `json:"Period"`
	EvaluationPeriods                int                                      `json:"EvaluationPeriods"`
	ComparisonOperator               string                                   `json:"ComparisonOperator"`
	Threshold                        float64                                  `json:"Threshold"`
	TreatMissingData                 string                                   `json:"TreatMissingData"`
	EvaluateLowSampleCountPercentile string                                   `json:"EvaluateLowSampleCountPercentile"`
}

type CloudWatchAlermMessageTriggerDimension struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

func (m *CloudWatchAlermMessage) IsRedshiftServerlessLimitUsage() bool {
	return m.Trigger.MetricName == "UsageLimitAvailable" && m.Trigger.Namespace == "AWS/Redshift-Serverless"
}

func (m *CloudWatchAlermMessage) WorkgroupName() string {
	for _, d := range m.Trigger.Dimensions {
		if strings.EqualFold(d.Name, "Workgroup") {
			return d.Value
		}
	}
	return ""
}

func (m *CloudWatchAlermMessage) UsageLimitID() string {
	for _, d := range m.Trigger.Dimensions {
		if strings.EqualFold(d.Name, "UsageLimitId") {
			return d.Value
		}
	}
	return ""
}
