# 🤖 QUY CHUẨN AI AGENT WORKSPACE (.agents)

**Dự án:** Rent-a-Girlfriend Platform  
**Mục tiêu:** Giới thiệu tổng quan về thư mục quản lý tri thức và hành vi của AI Coding Assistant (`.agents`).

---

## 1. TỔNG QUAN

Thư mục `.agents` ở gốc của dự án là nơi lưu trữ các chỉ dẫn cấu hình nghiệp vụ, kiến trúc, và các bộ kỹ năng (skills) bổ trợ dành riêng cho các **AI Coding Assistants** (như Antigravity, Cline, Windsurf...) khi tham gia pair-programming trong dự án.

Việc đóng gói các tài liệu này trong Git Repository giúp:
*   **Nhất quán tuyệt đối:** Mọi AI Agent tham gia phát triển đều tự động tuân thủ cùng một bộ tiêu chuẩn thiết kế kiến trúc và quy chuẩn nghiệp vụ.
*   **Tránh lệch lạc kiến trúc:** Đảm bảo code sinh ra luôn khớp với thiết kế Hexagonal, DDD, và các nguyên tắc truyền thông bất đồng bộ của hệ thống.
*   **Tự động hóa nâng cao:** Cung cấp các bộ kỹ năng chuyên biệt để AI có thể tự động thực hiện các tác vụ phức tạp một cách chuẩn chỉ.

---

## 2. CẤU TRÚC THƯ MỤC `.agents`

Thư mục `.agents` được chia thành 2 phần cốt lõi rất dễ đọc:

### 2.1. Hệ thống luật lệ (`.agents/rules/`)
Chứa các quy tắc nền tảng mà AI **bắt buộc** phải tuân thủ trong suốt quá trình code:
*   [agent-behavior.md](../../.agents/rules/agent-behavior.md): Quy chuẩn hành vi tư duy trước khi code, chỉnh sửa tối thiểu và xác thực kết quả.
*   [core-philosophy.md](../../.agents/rules/core-philosophy.md): Các triết lý thiết kế lõi như DDD, Hexagonal Architecture và cấm kết nối chéo Database.
*   [ddd-business-logic-conventions.md](../../.agents/rules/ddd-business-logic-conventions.md): Ngôn ngữ chung (Ubiquitous Language) và quy tắc Snapshot thông số giao dịch.
*   [distributed-communication-patterns.md](../../.agents/rules/distributed-communication-patterns.md): Chuẩn Service Mesh (Istio Ambient), xác thực JWT tập trung, và chuẩn API/Event.
*   [knowledge-persistence.md](../../.agents/rules/knowledge-persistence.md): Cách thức cập nhật và đề xuất đóng góp luật mới khi phát hiện tri thức quan trọng.
*   [universal-architecture-standards.md](../../.agents/rules/universal-architecture-standards.md): Cấu trúc thư mục chi tiết cho các Go/Node service, dependency flow, và yêu cầu tự viết tài liệu/test.

### 2.2. Bộ kỹ năng bổ trợ (`.agents/skills/`)
Chứa các bộ công cụ và tài liệu hướng dẫn giúp AI thực hiện các tác vụ tự động hóa chuyên biệt:
*   [git-commit](../../.agents/skills/git-commit/SKILL.md): Kỹ năng phân tích mã nguồn và tự động tạo thông điệp Git Commit theo chuẩn Conventional Commits.
*   [protobuf-standards](../../.agents/skills/protobuf-standards/SKILL.md): Quy chuẩn soạn thảo, review và đảm bảo tính tương thích ngược cho các tệp Protobuf.

---

> [!TIP]
> **Dành cho Lập trình viên:** Tất cả tài liệu trong `.agents` đều được viết dưới định dạng Markdown `.md` vô cùng tường minh. Bạn có thể mở trực tiếp các file cụ thể được liên kết phía trên, hoặc tự do khám phá các thư mục này trực tiếp thông qua thanh công cụ Explorer của IDE (VS Code, Cursor, v.v.).
