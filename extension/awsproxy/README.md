# AWS Proxy
<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [beta]  |
| Distributions | [contrib], [aws], [sumo] |
| Issues        | ![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aextension%2Fawsproxy%20&label=open&color=orange&logo=opentelemetry) ![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aextension%2Fawsproxy%20&label=closed&color=blue&logo=opentelemetry) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@Aneurysm9](https://www.github.com/Aneurysm9), [@mxiamxia](https://www.github.com/mxiamxia) |

[beta]: https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[aws]: https://github.com/aws-observability/aws-otel-collector
[sumo]: https://github.com/SumoLogic/sumologic-otel-collector
<!-- end autogenerated section -->


The AWS proxy accepts requests without any authentication of AWS signatures applied and forwards them to the
AWS API, applying authentication and signing. This allows applications to avoid needing AWS credentials to access
a service, instead configuring the AWS exporter and/or proxy in the OpenTelemetry collector and only providing the
collector with credentials.

## Configuration

Example:

```yaml
extensions:
  awsproxy:
    endpoint: 0.0.0.0:2000
    proxy_address: ""
    tls:
      insecure: false
      server_name_override: ""
    region: ""
    role_arn: ""
    aws_endpoint: ""
    local_mode: false
```

### endpoint (Optional)
The TCP address and port on which this proxy listens for requests.

Default: `0.0.0.0:2000`

### proxy_address (Optional)
Defines the proxy address that this extension forwards HTTP requests to the AWS backend through. If left unconfigured, requests will be sent directly.
This will generally be set to a NAT gateway when the collector is running on a network without public internet.

### insecure (Optional)
Enables or disables TLS certificate verification when this proxy forwards HTTP requests to the AWS backend. This sets the `InsecureSkipVerify` in the [TLSConfig](https://godoc.org/crypto/tls#Config). When setting to true, TLS is susceptible to man-in-the-middle attacks so it should be used only for testing.

Default: `false`

### server_name_override (Optional)
This sets the ``ServerName` in the [TLSConfig](https://godoc.org/crypto/tls#Config).

### region (Optional)
The AWS region this proxy forwards requests to. When missing, we will try to retrieve this value through environment variables or optionally ECS/EC2 metadata endpoint (depends on `local_mode` below).

### role_arn (Optional)
The IAM role used by this proxy when communicating with the AWS service. If non-empty, the receiver will attempt to call STS to retrieve temporary credentials, otherwise the standard AWS credential [lookup](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials) will be performed.

### aws_endpoint (Optional)
The AWS service endpoint which this proxy forwards requests to. If not set, will default to the AWS X-Ray endpoint.
