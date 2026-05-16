# ADR 0003: SSE Authentication Strategy & Istio Integration

## Trạng thái (Status)
Đã duyệt (Accepted)

## Bối cảnh (Context)
Hệ thống Notification Service cần duy trì kết nối Server-Sent Events (SSE) với Client. Vì hệ thống áp dụng kiến trúc **Auth Offloading** (sử dụng Istio Waypoint Proxy để tự động verify JWT Token ở tầng hạ tầng, trước khi request chạm tới code Golang), chúng ta cần xác định cách Client truyền JWT Token.

Tuy nhiên, API nguyên bản `EventSource` của trình duyệt Web **không hỗ trợ truyền Custom HTTP Headers** (như `Authorization: Bearer <token>`). Các dự án thường lách luật bằng cách truyền Token qua URL Query String (`?token=...`).

Việc truyền Token qua Query String mang lại rủi ro bảo mật lớn: Token sẽ bị ghi lại nguyên văn trong Access Logs của Load Balancer, Nginx, Istio, và lưu lại trong lịch sử trình duyệt.

## Quyết định (Decision)
1. **Tuân thủ tuyệt đối Auth Offloading:** Notification Service (Golang) sẽ **KHÔNG** tự parse JWT. Istio sẽ đảm nhiệm việc verify JWT từ header `Authorization: Bearer <token>`. Nếu hợp lệ, Istio bóc tách `userId` và truyền vào header nội bộ (VD: `user-id`) cho Golang.
2. **Cấm truyền Token qua URL:** Tuyệt đối không chấp nhận truyền JWT qua Query String đối với các kết nối SSE.
3. **Yêu cầu Frontend:** Để khắc phục giới hạn của API trình duyệt, Team Frontend Web **BẮT BUỘC** phải sử dụng thư viện mô phỏng SSE dựa trên Fetch API (Ví dụ: `@microsoft/fetch-event-source`). Thư viện này cho phép truyền Custom Headers đầy đủ. (Frontend Mobile không bị ảnh hưởng vì các thư viện HTTP của Mobile luôn hỗ trợ truyền header cho stream).

## Hệ quả (Consequences)
- **Tích cực:** 
  - Bảo mật tối đa. JWT không bao giờ bị lộ trong hệ thống Logs (Access Logs).
  - Code Golang cực kỳ sạch và nhẹ, không phải lo logic mã hóa/giải mã rườm rà.
  - Phù hợp với kiến trúc Mesh (Zero Trust).
- **Tiêu cực:** 
  - Đội ngũ Frontend Web không thể dùng code "mì ăn liền" (native `new EventSource()`) mà bắt buộc phải import thêm một thư viện ngoài. Tuy nhiên, đánh đổi này là hoàn toàn xứng đáng cho môi trường Production.
