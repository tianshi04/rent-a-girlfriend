package com.rentagf.notification.interfaces.http.filter;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletRequestWrapper;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.util.*;

/**
 * Bộ lọc Mock Xác thực chỉ kích hoạt ở môi trường phát triển (profile local hoặc dev).
 * Tự động chèn header 'user-id' mặc định nếu không có Istio Waypoint Proxy đứng trước để bypass security.
 */
@Component
@Profile({"local", "dev"})
public class MockAuthFilter extends OncePerRequestFilter {

    private static final Logger log = LoggerFactory.getLogger(MockAuthFilter.class);
    
    // Header xác thực chuẩn theo rule [distributed-communication-patterns.md]
    public static final String USER_ID_HEADER = "user-id";
    
    // UUID mock mặc định cho local testing
    public static final String DEFAULT_MOCK_USER_ID = "123e4567-e89b-12d3-a456-426614174000";

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain filterChain)
            throws ServletException, IOException {

        String userIdHeader = request.getHeader(USER_ID_HEADER);

        if (userIdHeader == null || userIdHeader.isBlank()) {
            log.debug("No 'user-id' header found in request. Auto-mocking with: {}", DEFAULT_MOCK_USER_ID);
            
            // Bọc request để ghi đè header
            HttpServletRequest wrappedRequest = new MockAuthRequestWrapper(request, USER_ID_HEADER, DEFAULT_MOCK_USER_ID);
            filterChain.doFilter(wrappedRequest, response);
        } else {
            filterChain.doFilter(request, response);
        }
    }

    /**
     * Request Wrapper tùy chỉnh để bổ sung header 'user-id' một cách động.
     */
    private static class MockAuthRequestWrapper extends HttpServletRequestWrapper {
        private final String headerName;
        private final String headerValue;

        public MockAuthRequestWrapper(HttpServletRequest request, String headerName, String headerValue) {
            super(request);
            this.headerName = headerName;
            this.headerValue = headerValue;
        }

        @Override
        public String getHeader(String name) {
            if (headerName.equalsIgnoreCase(name)) {
                return headerValue;
            }
            return super.getHeader(name);
        }

        @Override
        public Enumeration<String> getHeaderNames() {
            List<String> names = Collections.list(super.getHeaderNames());
            if (!names.contains(headerName)) {
                names.add(headerName);
            }
            return Collections.enumeration(names);
        }

        @Override
        public Enumeration<String> getHeaders(String name) {
            if (headerName.equalsIgnoreCase(name)) {
                return Collections.enumeration(Collections.singletonList(headerValue));
            }
            return super.getHeaders(name);
        }
    }
}
