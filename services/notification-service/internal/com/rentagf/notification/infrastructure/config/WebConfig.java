package com.rentagf.notification.infrastructure.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;
import org.springframework.web.filter.CorsFilter;

import java.util.List;

/**
 * Global CORS configuration for the Notification Service.
 *
 * <p>Allows the static HTML dev-tool (served via VS Code Live Server on 127.0.0.1:5500)
 * to connect to the backend running on localhost:8084.
 *
 * <p>In production, replace the hardcoded dev origins with values from application properties.
 */
@Configuration
public class WebConfig {

    /** Origins allowed in local development. */
    private static final List<String> DEV_ORIGINS = List.of(
            "http://127.0.0.1:5500",
            "http://localhost:5500",
            "http://localhost:8084",
            "http://127.0.0.1:8084"
    );

    @Bean
    public CorsFilter corsFilter() {
        CorsConfiguration config = new CorsConfiguration();

        // Explicit origins (required when credentials = true)
        config.setAllowedOrigins(DEV_ORIGINS);

        // Allow all standard HTTP methods
        config.setAllowedMethods(List.of("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"));

        // Allow all headers – includes custom 'user-id' header used by the SSE client
        config.addAllowedHeader("*");

        // Expose headers that the client may need to read
        config.setExposedHeaders(List.of("Content-Type", "Transfer-Encoding", "Cache-Control"));

        // Allow credentials (cookies, authorization headers)
        config.setAllowCredentials(true);

        // Cache preflight response for 1 hour
        config.setMaxAge(3600L);

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", config);

        return new CorsFilter(source);
    }
}
