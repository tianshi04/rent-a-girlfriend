{{- define "identity-service.fullname" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "identity-service.labels" -}}
helm.sh/chart: {{ include "identity-service.chart" . }}
app.kubernetes.io/name: {{ include "identity-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "identity-service.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "identity-service.name" -}}
{{- printf "%s" .Chart.Name -}}
{{- end -}}
