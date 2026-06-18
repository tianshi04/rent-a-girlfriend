{{- define "dispute-service.fullname" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dispute-service.labels" -}}
helm.sh/chart: {{ include "dispute-service.chart" . }}
app.kubernetes.io/name: {{ include "dispute-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "dispute-service.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "dispute-service.name" -}}
{{- printf "%s" .Chart.Name -}}
{{- end -}}
