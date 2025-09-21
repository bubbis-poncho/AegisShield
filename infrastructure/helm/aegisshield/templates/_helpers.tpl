{{/*
Expand the name of the chart.
*/}}
{{- define "aegisshield.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "aegisshield.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "aegisshield.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "aegisshield.labels" -}}
helm.sh/chart: {{ include "aegisshield.chart" . }}
{{ include "aegisshield.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "aegisshield.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aegisshield.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "aegisshield.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "aegisshield.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the PostgreSQL hostname
*/}}
{{- define "aegisshield.postgresql.host" -}}
{{- .Values.postgresql.host -}}
{{- end }}

{{/*
Get the Neo4j hostname
*/}}
{{- define "aegisshield.neo4j.host" -}}
{{- .Values.neo4j.host -}}
{{- end }}

{{/*
Get the Kafka brokers
*/}}
{{- define "aegisshield.kafka.brokers" -}}
{{- .Values.kafka.brokers -}}
{{- end }}

{{/*
Create image name for a service
*/}}
{{- define "aegisshield.image" -}}
{{- $registry := .registry | default .global.imageRegistry -}}
{{- $repository := .repository -}}
{{- $tag := .tag | default $.Chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s/%s:%s" $registry .global.repository $repository $tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .global.repository $repository $tag -}}
{{- end -}}
{{- end -}}

{{/*
Environment variables for database connection
*/}}
{{- define "aegisshield.databaseEnv" -}}
- name: DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.postgresql.existingSecret }}
      key: postgresql-url
- name: NEO4J_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.neo4j.existingSecret }}
      key: neo4j-url
{{- end }}

{{/*
Environment variables for Kafka connection
*/}}
{{- define "aegisshield.kafkaEnv" -}}
- name: KAFKA_BROKERS
  value: {{ include "aegisshield.kafka.brokers" . | quote }}
{{- end }}

{{/*
Common deployment template
*/}}
{{- define "aegisshield.deployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .name }}
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .name }}
    {{- include "aegisshield.labels" .context | nindent 4 }}
spec:
  replicas: {{ .config.replicaCount }}
  selector:
    matchLabels:
      app: {{ .name }}
      {{- include "aegisshield.selectorLabels" .context | nindent 6 }}
  template:
    metadata:
      labels:
        app: {{ .name }}
        {{- include "aegisshield.selectorLabels" .context | nindent 8 }}
    spec:
      serviceAccountName: {{ include "aegisshield.serviceAccountName" .context }}
      securityContext:
        {{- toYaml .context.Values.podSecurityContext | nindent 8 }}
      containers:
      - name: {{ .name }}
        securityContext:
          {{- toYaml .context.Values.securityContext | nindent 10 }}
        image: {{ include "aegisshield.image" (merge .config.image .context.Values.image (dict "global" .context.Values)) }}
        imagePullPolicy: {{ .context.Values.image.pullPolicy }}
        ports:
        - name: http
          containerPort: {{ .config.service.port }}
          protocol: TCP
        env:
        {{- include "aegisshield.databaseEnv" .context | nindent 8 }}
        {{- include "aegisshield.kafkaEnv" .context | nindent 8 }}
        {{- if .extraEnv }}
        {{- toYaml .extraEnv | nindent 8 }}
        {{- end }}
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          {{- toYaml .config.resources | nindent 10 }}
      {{- with .context.Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .context.Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .context.Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
{{- end }}

{{/*
Common service template
*/}}
{{- define "aegisshield.service" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ .name }}-service
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .name }}
    {{- include "aegisshield.labels" .context | nindent 4 }}
spec:
  type: {{ .config.service.type }}
  ports:
  - port: {{ .config.service.port }}
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app: {{ .name }}
    {{- include "aegisshield.selectorLabels" .context | nindent 4 }}
{{- end }}

{{/*
Common HPA template
*/}}
{{- define "aegisshield.hpa" -}}
{{- if .config.autoscaling.enabled }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ .name }}-hpa
  namespace: {{ .Release.Namespace }}
  labels:
    app: {{ .name }}
    {{- include "aegisshield.labels" .context | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .name }}
  minReplicas: {{ .config.autoscaling.minReplicas }}
  maxReplicas: {{ .config.autoscaling.maxReplicas }}
  metrics:
  {{- if .config.autoscaling.targetCPUUtilizationPercentage }}
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: {{ .config.autoscaling.targetCPUUtilizationPercentage }}
  {{- end }}
  {{- if .config.autoscaling.targetMemoryUtilizationPercentage }}
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: {{ .config.autoscaling.targetMemoryUtilizationPercentage }}
  {{- end }}
{{- end }}
{{- end }}