# ADR 0001: Chuẩn hóa công cụ phân tích tĩnh (Linting) và định dạng code (Formatting) trong CI Pipeline

**Trạng thái:** Accepted  
**Ngày:** 2026-05-31  

---

## Ngữ cảnh (Context)

Hệ thống **Rent-a-Girlfriend Platform** được thiết kế theo kiến trúc Microservices đa ngôn ngữ (Polyglot Microservices), bao gồm các dịch vụ viết bằng **Go**, **Rust**, **Python**, và **Java**. 

Do có sự tham gia của nhiều lập trình viên và các AI Coding Assistants khác nhau, dự án đối mặt với các vấn đề:
1.  **Thiếu nhất quán về phong cách viết code (Code Style):** Mỗi ngôn ngữ lập trình có những chuẩn viết code riêng, dẫn đến việc commit code có style lộn xộn, gây khó khăn cho việc Review Code (Pull Request).
2.  **Khó khăn trong phát hiện lỗi sớm:** Không có một bộ quy tắc chung để ngăn chặn các đoạn code kém chất lượng, có nguy cơ tiềm ẩn lỗi logic (ví dụ: biến không sử dụng, code không an toàn, cảnh báo compiler) được tích hợp vào nhánh chính.
3.  **Tối ưu hóa thời gian xây dựng (Build Time):** Các CI pipeline nếu không được chuẩn hóa linter và cơ chế lưu bộ nhớ đệm (caching) sẽ tốn rất nhiều thời gian thực thi, làm chậm nhịp phát triển của dự án.

Do đó, cần có một quyết định kỹ thuật thống nhất để lựa chọn và chuẩn hóa các công cụ linter/formatter cho từng ngôn ngữ trong CI Pipeline của dự án.

---

## Quyết định (Decision)

Chúng tôi quyết định chuẩn hóa các công cụ kiểm tra tĩnh (Linting), định dạng (Formatting) và các cấu hình nghiêm ngặt bắt buộc chạy trong CI Pipeline cho toàn bộ microservices như sau:

### 1. Chuẩn hóa công cụ theo ngôn ngữ:
*   **Với các dịch vụ Go (`booking-service`, `identity-service`):**
    *   Sử dụng **`gofmt`** làm định dạng code tiêu chuẩn.
    *   Sử dụng **`golangci-lint`** để quét và phân tích tĩnh mã nguồn.
*   **Với các dịch vụ Rust (`interaction-service`):**
    *   Sử dụng **`rustfmt`** làm công cụ định dạng code tiêu chuẩn.
    *   Sử dụng **`clippy`** làm linter chính, cấu hình nâng cao để chuyển toàn bộ cảnh báo (warnings) thành lỗi biên dịch nghiêm trọng.
*   **Với các dịch vụ Python (`profile-service`, `dispute-service`, `finance-service`):**
    *   Chuẩn hóa sử dụng công cụ hiệu năng cao **`ruff`** cho cả việc định dạng code và kiểm tra tĩnh, thay thế hoàn toàn cho sự kết hợp của các công cụ cũ.
*   **Với các dịch vụ Java (`notification-service`):**
    *   Sử dụng trình biên dịch Java tích hợp kết hợp với các bộ kiểm tra chuẩn của Gradle.

### 2. Nguyên tắc tích hợp trong CI Pipeline:
*   Các kiểm tra định dạng và phân tích tĩnh bắt buộc phải chạy ở giai đoạn đầu tiên của mỗi Pull Request trước khi tiến hành các bước kiểm thử.
*   Bất kỳ vi phạm nào về định dạng code hoặc cảnh báo linter đều bị coi là lỗi và chặn việc merge vào nhánh chính.
*   Tích hợp cơ chế lưu cache thư mục phụ thuộc của từng ngôn ngữ để tối ưu hóa thời gian phản hồi của pipeline.

---

## Hệ quả (Consequences)

### Điểm tích cực (Positives):
*   **Đồng nhất 100% Code Style:** Toàn bộ dự án có chất lượng code đồng đều, sạch sẽ và chuyên nghiệp, giúp quá trình Code Review diễn ra nhanh hơn.
*   **Phát hiện lỗi sớm:** Linter nghiêm ngặt giúp bắt được các lỗi bảo mật sơ đẳng, rò rỉ bộ nhớ hoặc các phản khuôn mẫu (anti-patterns) từ sớm mà không cần đợi chạy thử hay test thủ công.
*   **Hiệu năng CI vượt trội:** Việc lựa chọn các công cụ hiện đại và hiệu năng cao giúp giảm thời gian chạy CI của mỗi dịch vụ xuống dưới 2 phút, tiết kiệm tài nguyên hệ thống.
*   **Dễ dàng phối hợp giữa Người và AI:** AI Coding Assistants có thể tham chiếu trực tiếp đến tiêu chuẩn này để tự động sinh code chuẩn style ngay từ đầu.

### Đánh đổi (Negatives):
*   **Ràng buộc khắt khe đối với Developer:** Lập trình viên bắt buộc phải định dạng code và giải quyết triệt để các cảnh báo linter trước khi có thể tạo Pull Request thành công.
*   **Công sức bảo trì cấu hình ban đầu:** Cần duy trì các tệp cấu hình linter ở các thư mục dịch vụ để đảm bảo hoạt động đúng mong muốn.
