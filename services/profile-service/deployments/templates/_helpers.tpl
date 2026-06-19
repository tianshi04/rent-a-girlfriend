{{- define "profile-service.fullname" -}}
{{- printf "%s" .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "profile-service.labels" -}}
helm.sh/chart: {{ include "profile-service.chart" . }}
app.kubernetes.io/name: {{ include "profile-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "profile-service.chart" -}}
{{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "profile-service.name" -}}
{{- printf "%s" .Chart.Name -}}
{{- end -}}
