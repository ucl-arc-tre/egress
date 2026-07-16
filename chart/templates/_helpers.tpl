{{/*
HTTP container port
*/}}
{{- define "http_container_port" -}}
{{- 8000 }}
{{- end }}

{{/*
HTTPS container port
*/}}
{{- define "https_container_port" -}}
{{- 8443 }}
{{- end }}

{{/*
mTLS directory
*/}}
{{- define "mtls_dir" -}}
{{- "/etc/egress/tls" }}
{{- end }}

{{/*
generic storage mTLS directory
*/}}
{{- define "storage_generic_mtls_dir" -}}
{{- "/etc/egress/storage/generic/tls" }}
{{- end }}
