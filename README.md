# redshift-serverless-usage-limit-reporter

## Usage 

check [examples](./_examples) for sample Terraform code.
 
1. Apply terraform code and Deploy Lambda function.
1. Create a Redshift Serverless Usage Limit on Serverless Wrorkgroup.
1. Set SNS Topic as a notification destination for the Redshift Serverless Usage Limit.

### Slack Configuration.

manifest for SlackApp is as follows. 
```yaml
```yaml
display_information:
  name: redshift-serverless-usage-limit-reporter
  description: Redshift Serverless Usage Limit Reporter
  background_color: "#6f42c1"
features:
  app_home:
    home_tab_enabled: true
    messages_tab_enabled: false
    messages_tab_read_only_enabled: false
  bot_user:
    display_name: Redshift Serverless Usage Limit Reporter
    always_online: true
oauth_config:
  scopes:
    bot:
      - chat:write
```


Set SSM Parameter Store for `/prefix/SLACK_BOT_TOKEN`

lambda function environment variable `SSMWRAP_PREFIX` is `/prefix/`

## LICENSE

[MIT](./LICENSE)
