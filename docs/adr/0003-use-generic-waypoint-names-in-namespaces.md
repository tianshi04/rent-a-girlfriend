# ADR 0003: Sử dụng tên Waypoint Proxy cố định trong từng Namespace của Istio Ambient Mesh

**Trạng thái:** Accepted  
**Ngày:** 2026-06-21  

---

## Ngữ cảnh (Context)

Hệ thống **Rent-a-Girlfriend Platform** sử dụng **Istio Ambient Mesh** làm giải pháp Service Mesh thế hệ mới (Sidecar-less). Mặc định, ztunnel cung cấp cơ chế bảo mật truyền tải lớp L4 (mTLS, L4 Authorization). Tuy nhiên, để thực thi các chính sách bảo mật nâng cao lớp L7 (như kiểm tra JWT token qua `RequestAuthentication`, phân quyền chi tiết qua `AuthorizationPolicy` ở mức HTTP route), hệ thống bắt buộc phải sử dụng **Waypoint Proxy**.

Khi triển khai Waypoint Proxy để cô lập tài nguyên và chính sách L7 giữa các microservices (như `booking-service`, `identity-service`), chúng ta cần giải quyết các vấn đề sau:
1.  **Cách đặt tên cho Waypoint Proxy:** Chọn giữa việc đặt tên cố định giống nhau ở mọi namespace (Option A: `waypoint`) hay đặt tên động theo tên của dịch vụ (Option B: `<service-name>-waypoint`).
2.  **Phương pháp gán nhãn Namespace (Labeling):** Lựa chọn giữa việc gán nhãn thủ công qua dòng lệnh (`kubectl label`) hay khai báo trực tiếp trong các file cấu hình GitOps (Helm templates / Kustomize).
3.  **Cô lập và cách ly hạ tầng:** Xác định rõ phạm vi áp dụng Waypoint Proxy đối với các namespace nghiệp vụ so với các namespace hạ tầng (như `kafka`) và hệ thống quản trị (`kube-system`, `istio-system`, `flux-system`).

---

## Quyết định (Decision)

Chúng tôi quyết định chuẩn hóa mô hình triển khai Waypoint Proxy và quản lý cấu hình mạng trong Mesh như sau:

### 1. Chọn Option A: Tên Waypoint cố định (`waypoint`) cho từng Namespace nghiệp vụ
*   Mỗi namespace chứa microservice nghiệp vụ (như `booking-service`, `identity-service`) sẽ triển khai một Waypoint Proxy riêng với tên tài nguyên Gateway cố định là `waypoint`.
*   Cấu hình này được nhúng trực tiếp trong Helm chart của từng dịch vụ (`templates/istio/waypoint.yaml`):
    ```yaml
    apiVersion: gateway.networking.k8s.io/v1
    kind: Gateway
    metadata:
      name: waypoint
      namespace: {{ .Release.Namespace }}
    spec:
      gatewayClassName: istio-waypoint
    ```

### 2. Khai báo nhãn liên kết trực tiếp trong Git (Git-based Labeling)
*   Để liên kết namespace với Waypoint Proxy, nhãn `istio.io/use-waypoint: waypoint` phải được khai báo trực tiếp trong file định nghĩa namespace thuộc Helm chart (`templates/k8s/namespace.yaml`).
*   **Tuyệt đối nghiêm cấm** việc gán nhãn hoặc tạo Waypoint thủ công bằng lệnh CLI (`kubectl label namespace` hoặc `istioctl waypoint apply`) trên các môi trường persistent (Dev, Production). Mọi thay đổi cấu hình bắt buộc phải đi qua GitOps (FluxCD) để tránh hiện tượng trôi lệch cấu hình (Configuration Drift).

### 3. Chính sách cô lập Namespace hạ tầng & hệ thống:
*   **Namespace hạ tầng `kafka`:** Chỉ tham gia Mesh ở lớp L4 (ztunnel mTLS) bằng cách gắn nhãn `istio.io/dataplane-mode: ambient` để mã hóa đường truyền tin nhắn. **Tuyệt đối không** triển khai Waypoint Proxy (L7) cho namespace này vì Kafka sử dụng giao thức TCP binary, không tương thích với L7 proxy của Waypoint.
*   **Các namespace quản trị hệ thống (`kube-system`, `istio-system`, `flux-system`):** Nằm ngoài phạm vi Mesh hoàn toàn (không gắn nhãn Ambient, không chạy Waypoint) để đảm bảo độ tin cậy của các luồng điều khiển (control planes).

---

## Hệ quả (Consequences)

### Điểm tích cực (Positives):
*   **Tính mô-đun và tái sử dụng cao:** Cấu hình Helm templates cho Istio ở các microservices khác nhau là hoàn toàn giống nhau, giảm thiểu công sức cấu hình lặp lại.
*   **An toàn và Nhất quán:** Sử dụng GitOps làm Single Source of Truth (SSOT) đảm bảo cluster luôn tự động khôi phục cấu hình mesh chính xác khi xảy ra sự cố mà không phụ thuộc vào các lệnh gõ tay CLI.
*   **Hiệu năng và Bảo mật cân bằng:** Giữ Kafka ở lớp L4 đảm bảo tốc độ truyền tin cực nhanh (zero-overhead L7 parsing) trong khi vẫn duy trì kết nối an toàn mTLS.

### Đánh đổi (Negatives):
*   **Tăng context khi debug ở mức Cluster:** Khi chạy lệnh xem danh sách Gateway trên toàn cụm (`kubectl get gtw -A`), quản trị viên sẽ thấy nhiều Gateway có cùng tên là `waypoint`. Phải lọc theo cột `NAMESPACE` để xác định chính xác dịch vụ cần debug.
