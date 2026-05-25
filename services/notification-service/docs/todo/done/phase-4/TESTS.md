# 🧪 KẾ HOẠCH KIỂM THỬ NÂNG CẤP (TESTS.md) - PHASE 4: REST API

Tài liệu này đóng vai trò là **Mục lục Kiểm thử (Test Directory)** và **Kịch bản Kiểm thử chi tiết (Test Specification)** cho Phase 4: REST API theo phương pháp **Test-Driven Development (TDD)**. 

---

## 🗺️ Mục lục các Lớp Kiểm thử

Các file test sẽ được cấu trúc chuẩn tương ứng với cấu trúc thư mục của dự án:
1. **Unit Tests (Domain & Application Layers)**:
   - `NotificationTest.java` (Domain logic)
   - `CursorCodecTest.java` (Domain logic - Kiểm thử JSON Serialization và Base64 decode/encode)
   - `FetchInboxServiceTest.java` (Application logic - phân trang cursor, eventual consistency)
   - `MarkAsReadServiceTest.java` (Application logic - Kiểm tra luồng Optimistic Update và Fallback Exists)
   - `MarkAllAsReadServiceTest.java` (Application logic - cập nhật bulk)
2. **Web MVC Tests (Interfaces Layer)**:
   - `NotificationControllerTest.java` (HTTP Endpoints, Http Headers, Exception mapping)
3. **Integration Tests (Infrastructure & Persistence Layers)**:
   - `NotificationRepositoryImplTest.java` (Kiểm tra câu lệnh SQL/JPA pagination thực tế, kiểm tra câu Update Row-level)
   - `NotificationRestApiIntegrationTest.java` (Kiểm thử tích hợp End-to-End từ REST Endpoint tới Database)

---

## 🎯 Chi tiết Kịch bản Kiểm thử (Test Cases Specification)

### 1. Domain Unit Tests

#### 1.1. `CursorCodecTest.java` [NEW]
Kiểm thử bộ mã hóa và giải mã cursor bằng JSON.
- **`testEncodeDecode_JSON_Success`**:
  - **Mục tiêu**: Đảm bảo mã hóa ra chuỗi Base64 và giải mã ngược lại thành Object `InboxCursor` chuẩn xác thông qua JSON parser.
- **`testDecode_InvalidBase64_ThrowsException`**:
  - **Kỳ vọng**: Ném `InvalidCursorException` khi chuỗi không phải Base64.
- **`testDecode_InvalidJSONStructure_ThrowsException`**:
  - **Kỳ vọng**: Ném `InvalidCursorException` khi giải mã ra chuỗi JSON nhưng thiếu key `createdAt` hoặc `id`.

---

### 2. Application Service Unit Tests

#### 2.1. `MarkAsReadServiceTest.java`
Kiểm tra luồng xử lý cực kỳ tối ưu của **Optimistic Update**.
- **`testMarkAsRead_OptimisticUpdate_Success`**:
  - **Mục tiêu**: Đánh dấu thành công ngay lần đầu update.
  - **Mock**: Repository `markSingleAsRead(id, userId, now)` trả về `1` (rows_affected = 1).
  - **Kỳ vọng**: Gọi duy nhất 1 tương tác với Repository, không ném exception.
- **`testMarkAsRead_IdempotentFallback_Success`**:
  - **Mục tiêu**: Request một tin đã đọc từ trước.
  - **Mock**: 
    - `markSingleAsRead` trả về `0`.
    - `existsByIdAndUserId` trả về `true`.
  - **Kỳ vọng**: Trả về success (Idempotent theo `[INV-N08]`), không gọi thêm query nào khác.
- **`testMarkAsRead_BolaAttackOrNotFound_ThrowsException`**:
  - **Mục tiêu**: Ngăn chặn người dùng A cố tình sửa trạng thái của người dùng B, đồng thời che giấu sự tồn tại của ID (BOLA Guard).
  - **Mock**: 
    - `markSingleAsRead` trả về `0`.
    - `existsByIdAndUserId` trả về `false`.
  - **Kỳ vọng**: Ném ra `NotificationNotFoundException` (trả về HTTP 404).

---

### 3. HTTP Controller Unit Tests (`NotificationControllerTest.java`)
- **`testFetchInbox_ApiContract_Success`**:
  - **Kỳ vọng**: Trả về `200 OK` với cấu trúc JSON khớp 100% đặc tả `api-contract.md`. Cursor sinh ra là chuỗi Base64 của cục JSON hợp lệ.

---

### 4. Integration Tests

#### 4.1. `NotificationRepositoryImplTest.java`
Kiểm thử tích hợp tầng cơ sở dữ liệu để kiểm tra tính chính xác của các câu lệnh SQL.
- **`testJpaRepository_CursorPagination_ORDER_BY_Invariant`**:
  - **Kỳ vọng**: Truy vấn phân trang bắt buộc trả về dữ liệu đúng thứ tự `ORDER BY created_at DESC, id DESC`.
- **`testJpaRepository_OptimisticUpdate_RowsAffected`**:
  - **Kỳ vọng**:
    - Truyền ID chưa đọc $\rightarrow$ Trả về rows_affected = 1.
    - Truyền lại ID vừa update $\rightarrow$ Trả về rows_affected = 0 (Do không pass điều kiện `read_at IS NULL`).
    - Truyền ID sai user $\rightarrow$ Trả về rows_affected = 0.
