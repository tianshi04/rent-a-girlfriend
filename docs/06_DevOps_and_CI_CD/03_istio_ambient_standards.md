# Chuẩn cấu hình Istio Ambient Mesh và Quản lý Namespace

Tài liệu này định nghĩa các tiêu chuẩn DevOps đối với việc tích hợp dịch vụ vào **Istio Ambient Mesh**, cấu hình các thành phần định tuyến và bảo mật lớp L4 (ztunnel)/L7 (Waypoint Proxy), và chính sách cô lập hạ tầng.

---

## 1. Nguyên tắc cấu hình qua Git (Declarative Configuration)

Mọi cấu hình liên quan đến Kubernetes Namespaces, Istio Gateways, Policies và Routes bắt buộc phải được khai báo bằng mã nguồn (Declarative Configuration) trong Git thông qua Helm charts hoặc Kustomize:
- **Tuyệt đối không** thực hiện gán nhãn thủ công qua lệnh CLI (`kubectl label namespace` hoặc `istioctl waypoint apply`) trên các môi trường persistent (Dev, Production).
- Cơ chế GitOps (FluxCD) chịu trách nhiệm tự động đồng bộ và sửa đổi các sai lệch (drift reconciliation) trên cluster dựa theo mã nguồn trong Git.

---

## 2. Tiêu chuẩn cấu hình Namespace nghiệp vụ (Business Namespaces)

Đối với các namespace chứa microservice nghiệp vụ (như `booking-service`, `identity-service`), cấu hình bao gồm hai phần:

### 2.1. Đăng ký tham gia Mesh (L4)
Namespace phải được gắn nhãn `istio.io/dataplane-mode: ambient` để ztunnel tự động mã hóa mTLS và kiểm soát kết nối ở lớp mạng L4.
Khai báo trong file `templates/k8s/namespace.yaml`:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Release.Namespace }}
  labels:
    istio.io/dataplane-mode: ambient
```

### 2.2. Khai báo Waypoint Proxy & Liên kết (L7)
- **Tên cố định:** Mọi Waypoint Proxy trong namespace nghiệp vụ phải có tên cố định là `waypoint`.
- **Liên kết namespace:** Gắn nhãn `istio.io/use-waypoint: waypoint` trực tiếp tại tài nguyên Namespace để chỉ định mọi traffic đi qua Waypoint.
- **Khai báo Waypoint Gateway:** Tạo file template `templates/istio/waypoint.yaml` trong Helm chart của từng service:
```yaml
{{- if .Values.namespace.istio.ambientEnabled }}
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: waypoint
  namespace: {{ .Release.Namespace }}
spec:
  gatewayClassName: istio-waypoint
{{- end }}
```

---

## 3. Chính sách cô lập Namespace hạ tầng & hệ thống (Isolation Policy)

Để bảo vệ tính ổn định và tối ưu hóa hiệu năng toàn cụm, các namespace đặc biệt phải tuân thủ chính sách cô lập sau:

### 3.1. Namespace hạ tầng (`kafka`)
- **Đặc điểm:** Kafka phục vụ truyền nhận tin nhắn hiệu năng cao, sử dụng giao thức TCP binary riêng biệt, không tương thích với bộ phân tích L7 (HTTP/gRPC) của Waypoint Proxy.
- **Cấu hình chuẩn:**
  - Được phép tham gia Ambient Mesh ở lớp L4 để bảo mật truyền tải (`istio.io/dataplane-mode: ambient` được khai báo tại [kafka-namespace.yaml](../../infra/k8s/base/kafka-namespace.yaml)).
  - **Tuyệt đối nghiêm cấm** triển khai tài nguyên `Gateway` lớp `istio-waypoint` hoặc gắn nhãn `istio.io/use-waypoint` trong namespace `kafka`.

### 3.2. Các Namespace quản trị hệ thống (`kube-system`, `istio-system`, `flux-system`)
- **Đặc điểm:** Quản lý tài nguyên lõi, điều khiển mạng và đồng bộ GitOps.
- **Cấu hình chuẩn:**
  - Loại trừ hoàn toàn khỏi Mesh. **Không** gắn nhãn `istio.io/dataplane-mode` hay cấu hình Waypoint Proxy dưới mọi hình thức để tránh xung đột định tuyến và làm gián đoạn luồng điều khiển (control planes).

---

## 4. Tóm tắt quy trình kiểm tra (Verification)

Khi triển khai hoặc chỉnh sửa cấu hình Mesh, AI Agent và Lập trình viên có thể sử dụng các lệnh sau để xác minh trên cụm:

1.  **Kiểm tra nhãn của Namespace:**
    ```bash
    kubectl get ns -L istio.io/dataplane-mode -L istio.io/use-waypoint
    ```
2.  **Xác minh Waypoint Gateway hoạt động:**
    ```bash
    kubectl get gateway -n <namespace>
    ```
3.  **Kiểm tra tính hợp lệ của cấu hình mesh bằng istioctl:**
    ```bash
    istioctl analyze -n <namespace>
    ```
