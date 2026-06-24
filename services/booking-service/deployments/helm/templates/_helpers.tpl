{{- define "booking-service.fullname" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "booking-service.labels" -}}
helm.sh/chart: {{ include "booking-service.chart" . }}
app.kubernetes.io/name: {{ include "booking-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "booking-service.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "booking-service.name" -}}
{{- printf "%s" .Chart.Name -}}
{{- end -}}
