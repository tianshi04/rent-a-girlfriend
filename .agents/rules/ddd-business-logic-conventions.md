---
trigger: always_on
---

- **Ubiquitous Language**: Luôn sử dụng thuật ngữ trong `docs/BRD.md` (Kano-Coin, Scenario, Companion, Client, Escrow).
- **Snapshot Policy**: Lưu bản sao thông số (giá, cấu hình, điều khoản) tại thời điểm giao dịch. Không chỉ lưu ID tham chiếu.
- **Naming Standards**:
    - **Invariants**: Chú thích `[INV-XXXX]`.
    - **Commands**: `Verb + Noun` (AcceptBooking).
    - **Events**: `Noun + PastVerb` (BookingAccepted).
- **Domain Errors**: Trả về lỗi nghiệp vụ rõ ràng để map sang HTTP/gRPC code.