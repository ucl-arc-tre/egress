# Helm chart

Helm chart for deploying the egress service. See [`values.yaml`](./values.yaml) for the full
list of configurable values.

## S3 authentication

The S3 storage backend supports two authentication modes:

### Static credentials

Provide an access key pair via chart values:

```yaml
storage:
  provider: s3
  s3:
    region: eu-west-2
    access_key_id: AKIA...
    secret_access_key: ...
```

### IAM Roles for Service Accounts (IRSA)

On EKS, an IAM role can be assumed by the pod via a projected service account token, removing
the need to manage long-lived AWS keys. To enable IRSA, leave `access_key_id` and
`secret_access_key` unset, enable ServiceAccount creation, and annotate it with the IAM role
ARN to assume:

```yaml
storage:
  provider: s3
  s3:
    region: eu-west-2

serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::<account-id>:role/<role-name>
```

When `serviceAccount.annotations` is non-empty, the chart sets
`automountServiceAccountToken: true` on the pod so the projected token is available to the AWS
SDK. Credential resolution then falls through to the SDK default chain (web identity token via
the configured role).

Prerequisites on the AWS side (to be configured on the client-side): an OIDC provider associated with
the EKS cluster and an IAM role with a trust policy that allows the service account
(`system:serviceaccount:<namespace>:<serviceAccount.name>`) to assume it. See the
[AWS IRSA documentation](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
for details.
