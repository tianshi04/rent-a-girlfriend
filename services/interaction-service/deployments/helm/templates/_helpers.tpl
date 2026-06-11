{{- define "interaction-service.fullname" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "interaction-service.labels" -}}
helm.sh/chart: {{ include "interaction-service.chart" . }}
app.kubernetes.io/name: {{ include "interaction-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "interaction-service.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "interaction-service.name" -}}
{{- printf "%s" .Chart.Name -}}
{{- end -}}
