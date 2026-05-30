# 📅 KẾ HOẠCH DI CHUYỂN EMAIL LÊN PRODUCTION (Production Email Migration Plan)

Tài liệu này vạch ra kế hoạch chi tiết để chuyển đổi cấu hình Email từ môi trường phát triển (sử dụng **Mailtrap Sandbox**) sang môi trường vận hành thực tế (**Gmail SMTP** hoặc các dịch vụ chuyên nghiệp như **AWS SES / SendGrid**) mà không cần thay đổi mã nguồn.

---

## 1. So sánh Cấu hình: Mailtrap (Staging) vs Production

| Thông số | Mailtrap Sandbox (DEV/STG) | Gmail SMTP (Môi trường Nhỏ) | AWS SES / SendGrid (Production Lớn) |
| :--- | :--- | :--- | :--- |
| **Mục đích** | Thử nghiệm cô lập, không gửi tới người nhận thật | Gửi mail thật, quy mô nhỏ (< 500 mail/ngày) | Gửi mail thật, quy mô lớn, độ tin cậy cực cao |
| **Độ tin cậy** | Chỉ hiển thị trong Web Inbox của Mailtrap | Có thể bị Google đánh dấu Spam nếu gửi spam | **Rất cao** (Nhờ cấu hình SPF, DKIM, DMARC) |
| **Bảo mật** | TLS Optional, User/Password thô | Bắt buộc SSL/TLS, dùng **App Password** | Bắt buộc TLS, dùng IAM Credential chuyên biệt |

---

## 2. Phương án 1: Sử dụng Gmail SMTP (Quy mô Nhỏ / Demo Production)
Gmail SMTP phù hợp cho các dự án demo hoặc giai đoạn đầu của Production với lượng mail gửi đi nhỏ hơn 500 mail/ngày.

### Bước 1: Tạo App Password trên tài khoản Google
1. Truy cập vào quản lý Tài khoản Google (Google Account) có địa chỉ email bạn muốn gửi tin.
2. Bật tính năng **Xác minh 2 bước (2-Step Verification)** (Bắt buộc).
3. Tìm kiếm mục **Mật khẩu ứng dụng (App Passwords)**.
4. Tạo một mật khẩu ứng dụng mới với tên ví dụ: `rentagf-notification-service`.
5. Google sẽ cấp một chuỗi ký tự mật khẩu gồm 16 ký tự (Ví dụ: `abcd efgh ijkl mnop`). Lưu lại chuỗi này.

### Bước 2: Cập nhật cấu hình trên Production
Bạn chỉ cần nạp các biến môi trường sau cho Pod/Container chạy trên Production:
```env
SPRING_MAIL_HOST=smtp.gmail.com
SPRING_MAIL_PORT=587
SPRING_MAIL_USERNAME=your-email@gmail.com
# Sử dụng mật khẩu ứng dụng 16 ký tự vừa tạo, tuyệt đối không dùng mật khẩu chính
SPRING_MAIL_PASSWORD=abcdefghijklmnop
SPRING_MAIL_PROPERTIES_MAIL_SMTP_AUTH=true
SPRING_MAIL_PROPERTIES_MAIL_SMTP_STARTTLS_ENABLE=true
```

---

## 3. Phương án 2: Sử dụng AWS SES (Khuyến nghị cho Production Thực tế)
AWS Simple Email Service (SES) cung cấp hiệu năng vượt trội, chi phí cực rẻ ($0.1 cho 1000 email) và khả năng kiểm soát độ tin cậy tuyệt đối.

### Bước 1: Xác thực Tên miền (Domain Verification) trên AWS
Để có thể gửi mail bằng một hòm thư có đuôi tên miền riêng (Ví dụ: `no-reply@rentagf.com`), bạn phải xác thực quyền sở hữu tên miền đó trong AWS Console.
1. Truy cập AWS SES -> **Verified Identities** -> **Create Identity**.
2. Chọn loại **Domain** và nhập tên miền `rentagf.com`.
3. AWS SES sẽ cung cấp các bản ghi **CNAME** phục vụ cho **DKIM** (DomainKeys Identified Mail).
4. Truy cập trang quản trị DNS của nhà đăng ký tên miền (Ví dụ: Cloudflare, GoDaddy) và cấu hình các bản ghi CNAME đó.

### Bước 2: Thiết lập SPF và DMARC chống giả mạo & tăng độ tin cậy (Deliverability)
Thêm các bản ghi DNS sau để các máy chủ email nhận (như Gmail, Outlook) không ném email của bạn vào hộp thư rác (Spam):
*   **Bản ghi SPF (TXT Record)**:
    *   *Name*: `@` hoặc trống.
    *   *Value*: `v=spf1 include:amazonses.com ~all` (Tuyên bố AWS SES được quyền gửi mail cho tên miền này).
*   **Bản ghi DMARC (TXT Record)**:
    *   *Name*: `_dmarc.rentagf.com`
    *   *Value*: `v=DMARC1; p=quarantine; pct=100; rua=mailto:dmarc-reports@rentagf.com` (Yêu cầu gom thư nghi ngờ giả mạo vào thư rác).

### Bước 3: Cấu hình ứng dụng
Sau khi tạo **SMTP Credentials** chuyên biệt trên AWS SES Console, cấu hình ứng dụng Production với các biến môi trường:
```env
SPRING_MAIL_HOST=email-smtp.us-east-1.amazonaws.com # Thay thế bằng AWS Region của bạn
SPRING_MAIL_PORT=587
SPRING_MAIL_USERNAME=AKIAIOSFODNN7EXAMPLE # SMTP Username do AWS cấp
SPRING_MAIL_PASSWORD=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY # SMTP Password do AWS cấp
SPRING_MAIL_PROPERTIES_MAIL_SMTP_AUTH=true
SPRING_MAIL_PROPERTIES_MAIL_SMTP_STARTTLS_ENABLE=true
```

---

## 4. Quản lý Chuyển đổi Không Gián đoạn (Zero-Downtime Rollout)
1. **Duy trì Môi trường Kép**: Giữ nguyên Mailtrap trên môi trường `staging` và `dev`.
2. **Profile-driven Configuration**:
   * Sử dụng file `application-dev.yml` / `application-staging.yml` chứa cấu hình Mailtrap.
   * Sử dụng file `application-prod.yml` (hoặc nạp động qua Kubernetes ConfigMap/Secret) chứa cấu hình Gmail/AWS SES.
3. Kích hoạt đúng môi trường bằng tham số khởi chạy Java:
   `java -jar app.jar --spring.profiles.active=prod`
