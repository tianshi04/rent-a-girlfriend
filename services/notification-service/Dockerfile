# ---- Stage 1: Build ----
FROM eclipse-temurin:21-jdk-alpine AS builder

WORKDIR /app

# Copy gradle files
COPY gradlew gradlew.bat ./
COPY gradle ./gradle

# Copy build config
COPY build.gradle settings.gradle ./

# Ensure gradlew has execute permission and pre-download gradle dependencies
RUN chmod +x gradlew
RUN ./gradlew dependencies --no-daemon || true

# Copy source and resource files
COPY internal ./internal
COPY resources ./resources

# Build the application bootable JAR
RUN ./gradlew bootJar --no-daemon -x test

# ---- Stage 2: Runtime ----
FROM eclipse-temurin:21-jre-alpine AS runtime

# Install system dependencies (certificates, timezone data)
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root system user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
RUN chown appuser:appgroup /app

# Copy only the bootable JAR file from the builder stage
COPY --from=builder --chown=appuser:appgroup /app/build/libs/*-SNAPSHOT.jar app.jar

# Switch to the non-root user
USER appuser

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
