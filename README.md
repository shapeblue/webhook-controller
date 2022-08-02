# webhook-controller

This controller connects is registered as a repository level webhook against respective repos. A constant set of commands are maintained for each repository. Data such as github access tokens, message broker endpoint , username and password are exported as env variables

```
GITHUB_ACCESS_TOKEN="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"
GITHUB_WEBHOOK_SECRET="HITCHHIKE"
RABBITMQ_USERNAME="admin"
RABBITMQ_PASSWORD="password"
RABBITMQ_ENDPOINT="10.1.33.108:5672"
```