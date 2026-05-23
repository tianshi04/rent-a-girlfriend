# 🧱 STRUCTURAL DESIGN PATTERNS

## 1. Adapter Pattern (Mẫu Phối hợp / Bộ chuyển đổi)

*   **Dẫn chứng trong dự án**:
    *   **Port (Interface)**: [NotificationRepository.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/repository/NotificationRepository.java) (Tầng Domain)
    *   **Adapter (Implementation)**: [NotificationRepositoryImpl.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/persistence/jpa/NotificationRepositoryImpl.java) (Tầng Infrastructure JPA)
    *   **Mapper**: [NotificationMapper.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/persistence/jpa/mapper/NotificationMapper.java) (Chuyển đổi dữ liệu cấu trúc)

### 📝 Tại sao sử dụng và hoạt động thế nào?
*   **Vấn đề**: Trong Hexagonal Architecture, lớp Domain (Core Business Logic) phải hoàn toàn cô lập, thuần khiết và không được phụ thuộc trực tiếp vào công nghệ lưu trữ dữ liệu (PostgreSQL, JPA/Hibernate) để tránh rủi ro thay đổi công nghệ ảnh hưởng tới nghiệp vụ. Tuy nhiên, tầng Application/Domain vẫn cần tương tác lưu trữ thông báo.
*   **Giải pháp**: 
    1.  Tầng Domain định nghĩa một **Port (Interface)** là `NotificationRepository` thể hiện nhu cầu nghiệp vụ (như `save`, `findById`, `findByUserId`).
    2.  Tầng Infrastructure (Persistence) triển khai một **Adapter** là `NotificationRepositoryImpl` thực hiện interface Port đó. Adapter này sử dụng Spring Data JPA (`NotificationJpaRepository`) để thao tác trực tiếp với Database.
    3.  Để cách ly hoàn toàn kiểu dữ liệu, Adapter sử dụng `NotificationMapper` để dịch (map) thực thể nghiệp vụ thuần Java (`Notification`) sang thực thể quản lý của JPA (`NotificationJpaEntity`) trước khi lưu xuống DB, và ngược lại dịch thực thể JPA nhận được từ DB thành Domain Model để trả về cho Domain Core.

### 🔑 Các đặc tính thiết kế thể hiện qua cặp Port - Adapter này:
*   **`NotificationRepository` (Interface ở Domain) thể hiện:**
    *   **Tính Trừu tượng (Abstraction):** Chỉ tập trung định nghĩa nghiệp vụ cần những phương thức lưu trữ nào (*Cái gì cần làm* - ví dụ: `save`, `findByEventId`), hoàn toàn ẩn đi công nghệ lưu trữ thực tế bên dưới.
    *   **Nguyên lý Đảo ngược Phụ thuộc (Dependency Inversion - chữ D trong SOLID):** Giúp tầng Domain (Core) không phụ thuộc trực tiếp vào Database (tầng thấp). Thay vào đó, tầng Database phải phụ thuộc ngược lại vào định nghĩa Interface của tầng Domain.
*   **`NotificationRepositoryImpl` (Class ở Infrastructure) thể hiện:**
    *   **Tính Đa hình (Polymorphism):** Cho phép hoán đổi bộ lưu trữ dễ dàng. Ví dụ: chạy thật thì nạp `JPA Implementation`, chạy Unit Test thì nạp `Mock/Memory Implementation` mà code nghiệp vụ ở Domain không cần sửa một dòng nào.
    *   **Tính Đóng gói (Encapsulation):** Đóng gói kín kẽ tất cả chi tiết công nghệ (Spring Data JPA, SQL, các thư viện ánh xạ dữ liệu `NotificationMapper`) bên trong lớp này. Bên ngoài Domain gọi đến sẽ không nhìn thấy và không cần biết Database bên dưới đang hoạt động ra sao.

### 📈 Ưu điểm đạt được
1.  **Tính Độc lập Công nghệ (Hexagonal Architecture)**: Tầng Domain hoàn toàn không biết gì về JPA, Spring Data, hay PostgreSQL. Nếu tương lai thay đổi database từ PostgreSQL sang MongoDB, ta chỉ việc viết một Adapter mới thực thi Port `NotificationRepository` mà không cần sửa bất kỳ dòng code nào trong Domain Core.
2.  **Cách ly Rủi ro**: Lỗi hoặc thay đổi cấu trúc bảng cơ sở dữ liệu được xử lý trọn vẹn trong Adapter và JPA Entity, không bị lan truyền lên tầng nghiệp vụ.

