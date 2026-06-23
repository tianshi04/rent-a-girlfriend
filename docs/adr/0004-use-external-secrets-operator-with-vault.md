# ADR 0004: Sử dụng External Secrets Operator (ESO) kết hợp với HashiCorp Vault ngoài cụm để quản lý Secrets

**Trạng thái:** Superseded by [ADR 0006](0006-use-native-kubernetes-secrets.md)  
**Ngày:** 2026-06-22  

---

## Ngữ cảnh (Context)

Hệ thống **Rent-a-Girlfriend Platform** triển khai trên Kubernetes và áp dụng mô hình GitOps (FluxCD). Hiện tại, các microservices quản lý biến môi trường nhạy cảm (secrets) bằng cách:
1. Định nghĩa tệp tin `.env.example` và sao chép thủ công thành `.env` để chạy ở môi trường local.
2. Định nghĩa các tệp `values.secret.yaml.example` trong cấu hình Helm Chart, hướng dẫn lập trình viên tạo file `values.secret.yaml` (đã gitignored) chứa plaintext secrets rồi truyền vào lệnh Helm install hoặc CI/CD pipeline.

Cách tiếp cận này gặp phải các vấn đề về bảo mật và vận hành trong GitOps:
* **Không đồng bộ với GitOps:** Do FluxCD kéo cấu hình trực tiếp từ Git Repository để đồng bộ tự động, việc không commit secrets lên Git khiến FluxCD không thể tự triển khai hoàn chỉnh ứng dụng nếu thiếu secrets, hoặc buộc phải sử dụng các cơ chế chèn ngoài luồng (push model) gây xung đột cấu hình.
* **Nguy cơ lộ lọt thông tin nhạy cảm:** Bất kỳ sai sót nào dẫn đến việc commit plaintext secrets lên Git đều để lại dấu vết vĩnh viễn trong lịch sử Git.
* **Thiếu khả năng kiểm soát tập trung:** Không có cơ chế xoay vòng tự động (rotation), ghi nhật ký truy cập (auditing) và phân quyền chặt chẽ cho từng dịch vụ đối với các secret.

Chúng tôi đã xem xét các phương án giải quyết:
1. **Mozilla SOPS:** Mã hóa file secret trong Git bằng khóa public/private (age/KMS). Ưu điểm là đơn giản, GitOps native. Nhược điểm là quản lý khóa private key phức tạp ở local và thiếu khả năng tích hợp/kiểm toán tập trung.
2. **Bitnami Sealed Secrets:** Mã hóa một chiều bằng khóa public của Controller trong cluster. Nhược điểm là rủi ro mất khóa private key của controller dẫn đến việc mất toàn bộ lịch sử mã hóa trong Git, đồng thời quy trình chỉnh sửa/nâng cấp secret thủ công rất phiền toái.
3. **External Secrets Operator (ESO) + Cloud Secret Manager / HashiCorp Vault:** Đồng bộ secret từ kho lưu trữ ngoài vào K8s Secret. Đảm bảo Git không lưu trữ bất kỳ dữ liệu nhạy cảm nào, hỗ trợ kiểm soát truy cập phân cấp và kiểm toán mạnh mẽ.

---

## Quyết định (Decision)

Chúng tôi quyết định áp dụng tiêu chuẩn quản lý secret tập trung cho toàn bộ hệ thống bằng giải pháp **External Secrets Operator (ESO)** kết hợp với **HashiCorp Vault ngoài cụm (External Vault)**:

### 1. Hạ tầng quản lý Secret
* Sử dụng **External Secrets Operator (ESO)** làm thành phần trung gian đồng bộ trong Kubernetes cluster.
* Sử dụng **HashiCorp Vault** (phiên bản tự vận hành ngoài cụm hoặc HCP Vault SaaS) làm Kho lưu trữ Secret trung tâm (Single Source of Truth cho Secrets).

### 2. Phương thức xác thực an toàn (Kubernetes Auth Method)
* Sử dụng phương thức xác thực **Kubernetes ServiceAccount** để kết nối ESO với Vault.
* Tuyệt đối **không** lưu trữ bất kỳ token tĩnh hoặc mật khẩu dài hạn nào của Vault bên trong Kubernetes cluster. Mỗi microservice sẽ sử dụng chính `ServiceAccount` của mình để tự xác thực và lấy secret thông qua cơ chế **Token Projection** của Kubernetes.

### 3. Cấu hình K8s Resources
* Triển khai duy nhất một tài nguyên **`ClusterSecretStore`** toàn cục để định cấu hình kết nối tới Vault ngoài cụm.
* Ở cấp độ từng microservice, sử dụng tài nguyên **`ExternalSecret`** để chỉ định rõ ràng các secret cần ánh xạ.

### 4. Cơ chế ánh xạ Explicit (Explicit Key Mapping)
* Áp dụng nguyên tắc **Ánh xạ từng Key riêng biệt (Explicit Mapping)** bằng cách chỉ rõ từng key cần kéo từ Vault và đặt tên cho key đó trong Kubernetes Secret.
* Tuyệt đối không sử dụng cơ chế đồng bộ hàng loạt (Bulk/Extract Mapping) trừ khi có sự đồng ý của Kiến trúc sư hệ thống, nhằm tuân thủ nguyên tắc Quyền hạn tối thiểu (Least Privilege).

### 5. Môi trường Local Development
* Đối với môi trường máy cá nhân (Local Development), lập trình viên tiếp tục sử dụng tệp tin `.env` cục bộ (được đưa vào `.gitignore`) để truyền các biến môi trường cấu hình/mock, tránh tạo sự phụ thuộc trực tiếp vào cụm Vault thật trong quá trình phát triển hàng ngày.

---

## Hệ quả (Consequences)

### Điểm tích cực (Positives):
* **Bảo mật cấp doanh nghiệp (Enterprise-Grade Security):** Dữ liệu nhạy cảm hoàn toàn sạch bóng khỏi Git history. Xác thực không cần mật khẩu thông qua định danh ServiceAccount giúp giảm thiểu nguy cơ lộ lọt thông tin.
* **Tích hợp GitOps mượt mà:** FluxCD chỉ khai báo các tệp Manifest của `ExternalSecret` (an toàn tuyệt đối để commit lên Git). ESO sẽ tự động giải quyết phần kéo secret thực tế về cho ứng dụng lúc runtime.
* **Dễ dàng quản lý & Bảo trì:** Khi thông tin Vault thay đổi, chỉ cần cập nhật tại một nơi duy nhất (`ClusterSecretStore`). Các ứng dụng hoàn toàn không bị ảnh hưởng.
* **Khả năng mở rộng:** Dễ dàng triển khai cho các môi trường Staging/Production khác nhau bằng cách trỏ đường dẫn Vault Path tương ứng.

### Đánh đổi (Negatives):
* **Độ phức tạp hạ tầng tăng:** Cần cài đặt và giám sát thêm External Secrets Operator trong Kubernetes cluster, đồng thời thiết lập và vận hành cụm HashiCorp Vault bên ngoài.
* **Phụ thuộc kết nối mạng:** Cụm Kubernetes bắt buộc phải có kết nối mạng thông suốt và an toàn tới địa chỉ của Vault để thực hiện đồng bộ.
