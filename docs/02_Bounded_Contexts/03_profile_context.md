# PROFILE & CATALOGUE CONTEXT

**Phân loại Subdomain:** Core Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Quản lý hồ sơ người dùng (Client và Companion), xây dựng và quản lý thương hiệu cá nhân để thu hút Client. Quản lý danh mục dịch vụ (Scenario) và các tài sản truyền thông (Media).

## 2. AGGREGATES & ENTITIES

### Aggregate Root 1: `UserProfile`
Chứa thông tin cá nhân của **mọi người dùng** (Client và Companion). Được tạo khi người dùng đăng ký lần đầu hoặc được tạo tự động từ sự kiện `identity.user-registered.v1`.

*   **State:** `UserId`, `DisplayName`, `Bio`, `Role` (CLIENT | COMPANION), `AvatarUrl`.
*   **Ghi chú:** `role` được thay đổi qua command `UpgradeProfileRole` (trigger bởi sự kiện identity khi admin phê duyệt). `AvatarUrl` chỉ là một URL raw, không qua presigned URL.

### Aggregate Root 2: `CompanionProfile`
Chứa thông tin **chỉ dành cho Companion** (tồn tại song song với `UserProfile` khi user có role `COMPANION`). Quản lý trạng thái và địa bàn hoạt động.

*   **State:** `CompanionId` (= `UserId`), `Status` (APPROVED | REJECTED), `AvailableCities`.
*   **Ghi chú:** Profiles được tạo với trạng thái `APPROVED` mặc định (không cần phê duyệt admin).

### Aggregate Root 3: `Scenario` (Kịch bản dịch vụ)
Mô tả thông tin gói dịch vụ mà Companion cung cấp, đảm bảo tính hợp lệ của giá cả và thời lượng.
*   **State:** `ScenarioId`, `CompanionId`, `Title`, `Description`, `Price`, `Duration` (phút), `Status` (ACTIVE, INACTIVE).

### Aggregate Root 4: `MediaAsset` (Tài sản truyền thông)
Quản lý file upload (đặc biệt là Voice Intro), kiểm soát dung lượng và định dạng.
*   **State:** `AssetId`, `CompanionId`, `AssetType` (IMAGE, VOICE), `Url`, `DurationSec`, `SizeBytes`, `Status` (PENDING, APPROVED, REJECTED).

## 3. VALUE OBJECTS
*   `Location`: Tỉnh/Thành phố hoạt động.
*   `Money`: Giá của kịch bản.
*   `MediaUrl`: Chuỗi URL hợp lệ đến External Storage (S3/Cloudinary).
*   `Duration`: Thời lượng kịch bản (được chuẩn hóa theo các block cố định: 60, 120, 180 phút).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-P01]` `Price` (Giá của kịch bản) luôn phải lớn hơn `0`.
*   `[INV-P02]` `Duration` phải nằm trong các mốc quy định của nền tảng (VD: 60, 120, 180 phút).
*   `[INV-P03]` Một Companion chỉ được tạo tối đa `N` Scenarios (VD: 5 kịch bản) để tránh spam.
*   `[INV-P04]` Nếu `AssetType` là VOICE, `DurationSec` không được vượt quá 30 giây.
*   `[INV-P05]` Nếu `AssetType` là VOICE, `SizeBytes` không được vượt quá 5MB.
*   `[INV-P06]` Nếu `AssetType` là IMAGE, `SizeBytes` không được vượt quá 2MB.

## 5. THIẾT KẾ COMMAND & EVENT

> **Lưu ý:** `CreateProfile` **không có REST endpoint**. Profile được tạo tự động bởi Kafka listener khi nhận sự kiện `identity.user-registered.v1` từ Identity Service (xem Section 9).

| Lệnh (Command) | Actor | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- | :--- |
| `CreateProfile` | Kafka Listener *(internal only)* | `userId`, `displayName`, `bio`, `role` | `ProfileCreated` { companionId, userId, displayName, availableCities } |
| `UpdateProfile` | Client / Companion | `userId`, `displayName` *(required)*, `bio`, `avatarUrl`, `availableCities` *(required)* | `ProfileUpdated` { companionId, displayName, bio, availableCities } |
| `PatchProfile` | Client / Companion | `userId`, bất kỳ subset của { `displayName`, `bio`, `avatarUrl`, `availableCities` } | `ProfileUpdated` { companionId, displayName, bio, availableCities } |
| `UpgradeProfileRole` | Kafka Listener *(internal only)* | `userId` | - (UserProfile.role -> COMPANION, tạo mới CompanionProfile nếu chưa có) |
| `CreateScenario` | Companion | `companionId`, `title`, `price`, `duration` | `ScenarioCreated` { scenarioId, companionId, price } |
| `RequestPresignedUrl` | Companion | `companionId`, `assetType`, `sizeBytes`, `durationSeconds`, `contentType` | - |
| `UploadVoiceIntro` | Companion | `companionId`, `fileUrl`, `duration`, `size` | `VoiceIntroUploaded` / `VoiceIntroRejected` { reason } |

## 6. READ MODELS (CQRS PROJECTIONS)

**Read Model: Companion Catalogue (Trang chủ tìm kiếm)**
*   **Cơ sở dữ liệu:** Elasticsearch hoặc Redis/MongoDB (MVP dùng PostgreSQL JOIN).
*   **Nguồn cập nhật:** Lắng nghe `ProfileUpdated`, `ScenarioCreated`, `ReviewSubmitted` (từ Interaction).
*   **Cấu trúc dữ liệu (JSON Document):**
    ```json
    {
      "companionId": "usr_123",
      "displayName": "Chizuru Mizuhara",
      "bio": "Hãy để tôi đóng vai một người bạn gái hoàn hảo...",
      "avatarUrl": "https://...",
      "role": "COMPANION",
      "averageRating": 4.9,
      "totalReviews": 150,
      "availableCities": ["Hanoi", "HCM"],
      "status": "APPROVED",
      "scenarios": [
         { "id": "scn_01", "title": "Hẹn hò rạp chiếu phim", "price": 500, "duration": 120, "status": "ACTIVE" }
      ],
      "voiceIntroUrl": "https://...(presigned, expires in 5min)"
    }
    ```
*   **Mục đích:** Cho phép Client tìm kiếm, lọc theo giá, thành phố, rating cực kỳ nhanh chóng mà không cần Join dữ liệu giữa Profile, Interaction và Booking.

## 7. DOMAIN SERVICES
*   `MediaValidationService`: Logic kiểm tra tính hợp lệ của File (VD: Voice intro không vượt quá 30s, dung lượng < 5MB).

## 8. REPOSITORIES
*   `IUserProfileRepository` — Lưu và truy vấn `UserProfile` (indexed by `user_id`).
*   `ICompanionProfileRepository` — Lưu và truy vấn `CompanionProfile` (indexed by `companion_id`). Hỗ trợ `search_approved` để lọc theo name, city, price.
*   `IScenarioRepository`
*   `IMediaAssetRepository`

## 9. INTEGRATION EVENTS CONSUMED

| Event | Source | Hành động trong Profile Service |
| :--- | :--- | :--- |
| `identity.user-registered.v1` | Identity Service | Tự động tạo `UserProfile` với role `CLIENT` nếu chưa có |
| `identity.user-role-upgraded.v1` (hoặc tương đương) | Identity Service | Gọi `UpgradeProfileRole`: nâng role lên `COMPANION`, tạo mới `CompanionProfile` |
