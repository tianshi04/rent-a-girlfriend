# SERVICE MESH VỚI ISTIO AMBIENT MODE

Tài liệu này chi tiết hóa cách thức hệ thống Rent-a-Girlfriend áp dụng **Istio Ambient Mode** để quản lý giao tiếp, bảo mật và quan sát hệ thống Microservices mà không cần sử dụng Sidecar proxy.

---

## 1. KIẾN TRÚC AMBIENT MODE (SIDE-CARLESS)

Ambient Mode là kiến trúc thế hệ mới của Istio, loại bỏ việc phải chạy một Envoy proxy (sidecar) bên trong từng Pod ứng dụng. Thay vào đó, nó chia mesh thành hai lớp:

### 1.1. Lớp Phủ Bảo Mật (Secure Overlay - L4)
*   **Thành phần:** `ztunnel` (Zero-trust tunnel).
*   **Cơ chế:** Chạy dưới dạng một DaemonSet trên mỗi Node trong cluster.
*   **Trách nhiệm:** 
    *   Tự động mã hóa mTLS cho mọi traffic giữa các service.
    *   Cung cấp định danh service (Service Identity) dựa trên SPIFFE.
    *   Thu thập L4 metrics (TCP bytes in/out, connection duration).
*   **Ưu điểm:** Cực kỳ nhẹ, không ảnh hưởng đến logic ứng dụng và không yêu cầu restart Pod khi cấu hình.

### 1.2. Lớp Xử Lý Nâng Cao (Layer 7 Processing)
*   **Thành phần:** `Waypoint Proxy`.
*   **Cơ chế:** Chạy dưới dạng một deployment riêng biệt (thường là một cái cho mỗi Namespace hoặc Service account). Traffic chỉ được điều hướng qua Waypoint khi cần xử lý mức L7.
*   **Trách nhiệm:** 
    *   **Traffic Management:** HTTP Routing, Retries, Circuit Breaking, Fault Injection.
    *   **Security:** RequestAuthentication (JWT Verification), AuthorizationPolicy mức chi tiết.
    *   **Observability:** HTTP metrics, distributed tracing (L7).

---

## 2. XÁC THỰC TẬP TRUNG VỚI JWT (OFFLOADING AUTH)

Một trong những ứng dụng quan trọng nhất của Istio trong dự án này là **xác thực JWT tập trung tại tầng Mesh**.

### 2.1. Vấn đề của cách làm truyền thống
Thông thường, mỗi Microservice (Booking, Finance, Profile...) phải tự cài đặt thư viện để parse JWT, verify chữ ký với JWKS công khai từ Identity Service. Việc này dẫn đến:
*   Lặp lại code ở nhiều nơi.
*   Khó cập nhật logic xác thực (ví dụ: đổi thuật toán mã hóa).
*   Tăng độ trễ cho ứng dụng.

### 2.2. Giải pháp với Istio Waypoint
Hệ thống sử dụng **Istio RequestAuthentication** để "thuê" mesh làm nhiệm vụ xác thực:
1.  **RequestAuthentication:** Cấu hình để Waypoint tự động lấy JWKS từ `Identity Service` và verify token trong header `Authorization: Bearer <JWT>`.
2.  **AuthorizationPolicy:** Chỉ cho phép các request đã được xác thực thành công (principal hợp lệ) đi vào service.
3.  **Hành vi:** Nếu token sai hoặc hết hạn, Istio sẽ trả về `401 Unauthorized` ngay tại Waypoint, request thậm chí không bao giờ chạm tới code của Microservice.

**Lợi ích:** Developer chỉ cần tập trung vào nghiệp vụ. Code của service mặc định coi như user đã được xác thực nếu request đến được tầng Application.

---

## 3. LỢI ÍCH TỔNG THỂ
*   **Tối ưu tài nguyên:** Tiết kiệm ~70% dung lượng proxy so với Sidecar.
*   **Tách biệt mối quan tâm (Separation of Concerns):** Hạ tầng lo bảo mật và giao tiếp, Service lo nghiệp vụ.
*   **Vận hành mượt mà:** Nâng cấp mesh mà không làm gián đoạn ứng dụng (Zero-downtime upgrades).
