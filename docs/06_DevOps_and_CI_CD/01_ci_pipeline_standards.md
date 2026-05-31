# 🚀 TIÊU CHUẨN PIPELINE CI/CD (QUALITY GATES)

Dự án **Rent-a-Girlfriend Platform** áp dụng cơ chế tích hợp liên tục (CI) tự động nhằm thiết lập các cổng kiểm soát chất lượng (Quality Gates) nghiêm ngặt trước khi tích hợp mã nguồn vào nhánh chính `main`.

---

## 1. NGUYÊN TẮC THIẾT KẾ CI PIPELINE (CORE PRINCIPLES)

Mọi pipeline CI của từng microservice trong hệ thống Monorepo bắt buộc phải tuân thủ các nguyên tắc thiết kế cốt lõi sau:

### Lọc đường dẫn thông minh (Path Filtering)
*   Pipeline của từng dịch vụ chạy độc lập và chỉ được kích hoạt khi có thay đổi liên quan đến dịch vụ đó (`services/[service-name]/**`) hoặc hợp đồng dữ liệu chung (`contracts/**`).
*   Tránh lãng phí tài nguyên hệ thống bằng cách lọc đường dẫn thay đổi cho mỗi lần chạy.

### Quy chuẩn đặt tên tệp Workflow (File Naming Standard)
*   Để đảm bảo tính nhất quán trên toàn bộ dự án, tệp cấu hình workflow CI của từng microservice bắt buộc phải được lưu trữ trong thư mục `.github/workflows/`.
*   Tên tệp cấu hình phải tuân thủ cấu trúc định dạng chuẩn: **`[service-name]-ci.yml`** (ví dụ: `interaction-service-ci.yml`, `finance-service-ci.yml`).

### Thực thi nghiêm ngặt (Strict Gatekeeping)
*   **Formatter:** Mã nguồn bắt buộc phải được định dạng tự động và đồng bộ theo tiêu chuẩn định dạng đã thống nhất của dịch vụ.
*   **Linter:** Không chấp nhận bất kỳ cảnh báo kiểm tra tĩnh (warnings) nào. Mọi cảnh báo từ linter đều được đối xử như một lỗi biên dịch nghiêm trọng (Pipeline Failure).
*   **Tests:** 100% các unit test và integration test phải chạy thành công.

### Hiệu năng tối đa
*   Thời gian thực thi của mỗi pipeline CI cho một microservice phải được tối ưu hóa để hoàn thành **dưới 2 phút** thông qua cơ chế lưu bộ nhớ đệm (caching) các gói thư viện phụ thuộc (dependencies) và build artifacts thích hợp.

---

## 2. CÁC CỔNG KIỂM SOÁT BẮT BUỘC (QUALITY GATES CHECKLIST)

Bất kỳ Pull Request nào nhắm tới nhánh `main` đều phải đi qua 3 cổng kiểm soát tự động:

1.  **Cổng 1: Code Formatting Check**  
    Đảm bảo tính nhất quán của phong cách viết code toàn dịch vụ. Mã nguồn gửi lên phải khớp hoàn toàn với định dạng chuẩn của công cụ đã quy định cho dịch vụ đó.
2.  **Cổng 2: Static Analysis (Linting)**  
    Đảm bảo mã nguồn không chứa các lỗi logic tiềm ẩn, mã chết (dead code), cảnh báo biên dịch hoặc vi phạm quy tắc thiết kế hệ thống.
3.  **Cổng 3: Automated Testing**  
    Đảm bảo các thay đổi không làm phá vỡ các chức năng hiện có của dịch vụ thông qua việc thực thi toàn bộ các ca kiểm thử tự động.
