---
trigger: always_on
---

- **Ghi nhận Quyết định (ADR)**: Chủ động tạo/cập nhật ADR khi thống nhất thiết kế kỹ thuật, kiến trúc, công nghệ (CSDL, thư viện, luồng chính...).
- **Vị trí Lưu trữ**:
    - **Cấp Service**: `services/<service-name>/docs/adr/` (ảnh hưởng cục bộ).
    - **Cấp Toàn cục**: `docs/adr/` ở thư mục root (ảnh hưởng toàn hệ thống/liên dịch vụ).
- **Quy tắc Đặt tên**: `XXXX-slug-ten-quyet-dinh.md` (Ví dụ: `0006-use-outbox-pattern.md`). Số `XXXX` tăng dần từ `0001`.
- **Cấu trúc ADR Tiêu chuẩn**:
    - `# ADR XXXX: [Tên quyết định]`
    - `**Trạng thái:** Accepted | Proposed | Rejected`
    - `**Ngày:** YYYY-MM-DD`
    - `## Ngữ cảnh (Context)`: Bối cảnh, ràng buộc kỹ thuật và phương án cân nhắc.
    - `## Quyết định (Decision)`: Giải pháp chọn và Rationale.
    - `## Hệ quả (Consequences)`: Điểm tích cực (Positives) và Đánh đổi (Negatives).
- **Quy trình Thực hiện**:
    - Soạn bản nháp ngay sau khi chốt phương án và đề xuất ghi file.
    - Không sửa ADR đã `Accepted`. Khi thay đổi, cập nhật ADR cũ sang `Superseded by ADR XXXX` và tạo ADR mới.
