# PROFILE & CATALOGUE CONTEXT

**Phân loại Subdomain:** Core Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Quản lý hồ sơ Companion, xây dựng và quản lý thương hiệu cá nhân để thu hút Client. Quản lý danh mục dịch vụ (Scenario) và các tài sản truyền thông (Media).

## 2. AGGREGATES & ENTITIES
### Aggregate Root 1: `CompanionProfile`
Chứa thông tin hiển thị của Companion.
*   **State:** Thông tin cá nhân, thành phố hoạt động, định danh người dùng.

### Aggregate Root 2: `Scenario` (Kịch bản dịch vụ)
Mô tả thông tin gói dịch vụ mà Companion cung cấp, đảm bảo tính hợp lệ của giá cả và thời lượng.
*   **State:** `ScenarioId`, `CompanionId`, `Title`, `Description`, `Price`, `Duration` (phút), `Status` (ACTIVE, INACTIVE).

### Aggregate Root 3: `MediaAsset` (Tài sản truyền thông)
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

## 5. THIẾT KẾ COMMAND & EVENT
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `CreateScenario` | `companionId`, `title`, `price`, `duration` | `ScenarioCreated` { scenarioId, companionId, price } |
| `UploadVoiceIntro` | `companionId`, `fileUrl`, `duration`, `size` | `VoiceIntroUploaded` / `VoiceIntroRejected` { reason } |

## 6. READ MODELS (CQRS PROJECTIONS)
**Read Model: Companion Catalogue (Trang chủ tìm kiếm)**
*   **Cơ sở dữ liệu:** Elasticsearch hoặc Redis/MongoDB.
*   **Nguồn cập nhật:** Lắng nghe `ProfileUpdated`, `ScenarioCreated`, `ReviewSubmitted` (từ Interaction).
*   **Cấu trúc dữ liệu (JSON Document):**
    ```json
    {
      "companionId": "cmp_123",
      "displayName": "Chizuru Mizuhara",
      "avatarUrl": "https://...",
      "averageRating": 4.9,
      "totalReviews": 150,
      "availableCities": ["Hanoi", "HCM"],
      "scenarios": [
         { "scenarioId": "scn_01", "title": "Hẹn hò rạp chiếu phim", "price": 500 }
      ],
      "voiceIntroUrl": "https://..."
    }
    ```
*   **Mục đích:** Cho phép Client tìm kiếm, lọc theo giá, thành phố, rating cực kỳ nhanh chóng mà không cần Join dữ liệu giữa Profile, Interaction và Booking.

## 7. DOMAIN SERVICES
*   `MediaValidationService`: Logic kiểm tra tính hợp lệ của File (VD: Voice intro không vượt quá 30s, dung lượng < 5MB).

## 8. REPOSITORIES
*   `ICompanionProfileRepository`
*   `IScenarioRepository`
*   `IMediaAssetRepository`
