{{- define "finance-service.fullname" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "finance-service.labels" -}}
helm.sh/chart: {{ include "finance-service.chart" . }}
app.kubernetes.io/name: {{ include "finance-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "finance-service.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "finance-service.name" -}}
{{- printf "%s" .Chart.Name -}}
{{- end -}}
