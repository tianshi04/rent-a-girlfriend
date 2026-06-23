# ADR 0006: Sử dụng Native Kubernetes Secrets (Opaque) thay thế cho External Secrets Operator

**Trạng thái:** Accepted  
**Ngày:** 2026-06-23  

---

## Ngữ cảnh (Context)

Trước đây, theo [ADR 0004](0004-use-external-secrets-operator-with-vault.md), hệ thống đã lựa chọn giải pháp **External Secrets Operator (ESO)** kết hợp với **HashiCorp Vault ngoài cụm** nhằm lưu trữ và đồng bộ hóa các biến môi trường nhạy cảm (secrets).

Tuy nhiên, trong quá trình phát triển thực tế và đánh giá lại hệ thống, chúng tôi nhận thấy:
1. **Độ phức tạp vận hành cao:** Vận hành một cụm Vault ngoài cùng với việc giám sát ESO trong cụm Kubernetes làm tăng đáng kể chi phí hạ tầng và thời gian quản trị.
2. **Khó khăn trong phát triển/kiểm thử:** Việc phụ thuộc vào một kết nối Vault trực tiếp lúc runtime làm cho quy trình kiểm thử cục bộ và thiết lập môi trường CI/CD bị chậm và phức tạp.
3. **Mã nguồn thực tế đã sẵn sàng cho K8s Secrets:** Các Helm Chart hiện tại của các dịch vụ cốt lõi (như `identity-service` và `booking-service`) đều đã được tích hợp sẵn tệp template `secret.yaml` tĩnh đọc từ `.Values.secrets` và chèn vào container thông qua `envFrom`. Việc này cho phép chúng ta chuyển sang K8s Secrets gốc ngay lập tức mà không cần bất kỳ thay đổi nào về code trong microservices.

Do đó, chúng tôi cần định nghĩa lại cơ chế quản lý secret để tối giản hóa kiến trúc và đảm bảo tính dễ dàng trong triển khai.

---

## Quyết định (Decision)

Chúng tôi quyết định thay thế hoàn toàn mô hình ESO + Vault bằng mô hình **Native Kubernetes Secrets (Opaque)** làm chuẩn quản lý secret chính thức cho toàn bộ hệ thống Rent-a-Girlfriend Platform:

1. **Sử dụng tài nguyên Secret gốc:** Sử dụng tài nguyên Kubernetes `kind: Secret` với kiểu dữ liệu `type: Opaque`.
2. **Khai báo Schema tĩnh qua Helm:** Trong tệp `values.yaml` của mỗi dịch vụ, các key secret bắt buộc phải được khai báo tường minh với giá trị rỗng (`""`) để làm mẫu (schema).
3. **Quy trình cung cấp secret an toàn (Secure Injection):**
   - **Local Development:** Sử dụng file `values.secret.yaml` (nằm trong `.gitignore`) để ghi đè các secret cục bộ khi chạy Helm install/upgrade offline.
   - **CI/CD & GitOps:** Khi triển khai lên cụm Staging/Production, các secret thật sẽ được lấy từ kho bí mật của CI/CD (ví dụ: GitHub Secrets) và nạp trực tiếp qua tham số `--set` hoặc sử dụng cơ chế giải mã file secret bằng SOPS được hỗ trợ trực tiếp bởi công cụ CD (như FluxCD HelmRelease decryption).
4. **Không thay đổi Deployment:** Giữ nguyên cơ chế mount secrets qua `envFrom` để tránh làm ảnh hưởng tới logic ứng dụng hiện tại.

---

## Hệ quả (Consequences)

### Điểm tích cực (Positives):
* **Tối giản hóa kiến trúc:** Loại bỏ hoàn toàn sự phụ thuộc vào ESO controller và cụm HashiCorp Vault. Giảm tải tài nguyên cho cụm Kubernetes.
* **Tương thích cao và độc lập:** Sử dụng trực tiếp chuẩn Kubernetes gốc, không phụ thuộc vào bất kỳ công cụ của bên thứ ba nào khi khởi chạy cơ bản.
* **Dễ dàng Debug offline:** Nhà phát triển có thể tự tạo nhanh các secret trên máy cá nhân mà không cần kết nối mạng đến Vault.

### Đánh đổi (Negatives):
* **Bảo mật GitOps đòi hỏi quy trình CI/CD tốt:** Vì không thể lưu plaintext secret trên Git, đội ngũ DevOps bắt buộc phải thiết lập quy trình CI/CD hoặc giải mã SOPS chặt chẽ để nạp secret vào cụm một cách an toàn.
* **Thiếu các tính năng nâng cao của Vault:** Hệ thống sẽ không có các tính năng xoay vòng khóa tự động (secret rotation) và ghi nhật ký truy cập (auditing) chi tiết như khi sử dụng HashiCorp Vault chuyên dụng. Tuy nhiên, mức độ bảo mật của K8s Secrets là hoàn toàn đủ đáp ứng cho quy mô hiện tại của dự án.
