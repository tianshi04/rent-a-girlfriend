# ADR 0002: Chuẩn hóa quy trình CI Build và Push Docker Image phục vụ luồng GitOps trên GHCR

**Trạng thái:** Accepted  
**Ngày:** 2026-05-31  

---

## Ngữ cảnh (Context)

Hệ thống **Rent-a-Girlfriend Platform** triển khai trên Kubernetes và áp dụng mô hình triển khai liên tục tự động **GitOps** trực tiếp từ nhánh chính `main`.

Để đảm bảo tính nhất quán, an toàn và bất biến tuyệt đối cho hệ thống GitOps, quy trình đóng gói container cần giải quyết các bài toán kiến trúc sau:
1.  **Tính bất biến tối cao (Zero Trust & Immutability):** Để tránh tuyệt đối các lỗi cache image hoặc việc đẩy đè tag, hệ thống GitOps bắt buộc phải chốt sử dụng **Image Digest (SHA256 hash gốc)** làm định danh triển khai duy nhất trong các tệp cấu hình K8s Manifests. Các thẻ tag động như `:latest` hoặc tag commit `:sha-[short-sha]` chỉ đóng vai trò tham chiếu nhanh bằng mắt thường và tra cứu Git lịch sử, tuyệt đối không dùng trực tiếp để chạy trên các môi trường.
2.  **Rủi ro Container lỗi khởi động (Crash Loop):** Việc Dockerfile build thành công và biên dịch hoàn tất không đảm bảo 100% container sẽ chạy được trong môi trường thật. Có nhiều trường hợp container bị thoát lập tức do thiếu cấu hình, sai thư viện động hoặc lỗi logic khởi động. Điều này làm cụm Kubernetes bị rơi vào trạng thái CrashLoopBackOff, gây gián đoạn hệ thống.
3.  **Tập trung hóa workflow:** Để tối giản hóa việc quản lý Monorepo, các tác vụ đóng gói Docker cần được tích hợp làm một job có tên `build-image` nằm trực tiếp trong tệp cấu hình CI hiện có của từng microservice (`.github/workflows/[service-name]-ci.yml`), thiết lập ràng buộc chỉ chạy sau khi mã nguồn đã vượt qua các cổng kiểm soát chất lượng (`lint-and-test`).

---

## Quyết định (Decision)

Chúng tôi quyết định áp dụng các chuẩn hóa sau đối với quy trình CI đóng gói Docker cho toàn bộ microservices phục vụ luồng GitOps:

### 1. Triển khai Bất biến bằng Image Digest (Digest-based GitOps)
*   **Chốt sử dụng Digest:** Trong các tệp triển khai GitOps chính thức, **bắt buộc chốt sử dụng duy nhất định danh Image Digest (`[image-path]@[digest-hash]`)** để kéo ảnh container về chạy.
*   **Ghi nhận Digest:** Cuối mỗi job `build-image` thành công, workflow bắt buộc phải thực hiện bước **in ra (output) mã băm Docker Image Digest (SHA256)** trong log của GitHub Actions run để phục vụ các script GitOps tự động hóa.
*   **Tag phụ trợ:** Image vẫn được push kèm các tag `:latest` và `:sha-[short-sha]` để phục vụ việc tra cứu lịch sử và kiểm thử thủ công nhanh.

### 2. Kiểm thử Khởi chạy Bắt buộc (Container Smoke Test)
*   **Kiểm thử khởi chạy:** Ngay sau khi build xong Docker image và trước khi push lên GHCR, CI runner bắt buộc phải thực hiện bước **Container Smoke Test** (dựng container từ image vừa build ở môi trường local).
*   **Xác thực tính sẵn sàng:** Thực hiện kiểm tra để đảm bảo container khởi chạy thành công, sẵn sàng xử lý yêu cầu và không bị tự động thoát hoặc crash.
*   **Tính linh hoạt:** Giải pháp kỹ thuật và kịch bản xác thực cụ thể (ví dụ: ping cổng lắng nghe, gọi API healthcheck, kiểm tra log khởi động) sẽ do từng microservice tự quyết định dựa trên đặc thù công nghệ, ngôn ngữ và giao thức của dịch vụ đó.
*   **Chặn tích hợp:** Bất kỳ lỗi khởi chạy hay crash nào trong quá trình Smoke Test đều được coi là một lỗi CI nghiêm trọng, lập tức chặn việc merge PR hoặc chặn push ảnh lên registry.

### 3. Quản lý Tập trung (Unified Workflows)
*   **Tích hợp chung:** Công việc build/push Docker được viết thành job `build-image` nằm trực tiếp trong tệp `.github/workflows/[service-name]-ci.yml` của từng microservice, sử dụng thuộc tính `needs: lint-and-test` để đảm bảo code sạch trước khi chạy.

### 4. Định cấu hình theo Git Event và Nhánh áp dụng
*   **Nhánh áp dụng:** Chỉ áp dụng cho nhánh **`main`** và các Pull Request nhắm tới `main`.
*   **Trong Pull Request (PR) nhắm tới `main`:**
    *   Chạy job `build-image` với tùy chọn `push: false`. Thực thi build và kiểm thử Container Smoke Test tại local CI runner. Không đẩy ảnh lên Registry.
*   **Trong sự kiện Push/Merge vào `main`:**
    *   Chạy job `build-image` with tùy chọn `push: true`. Thực thi build, Container Smoke Test và tiến hành push ảnh lên GHCR, xuất ra mã băm Image Digest ở cuối job.

---

## Hệ quả (Consequences)

### Điểm tích cực (Positives):
*   **An toàn và Bất biến tuyệt đối:** Image Digest bảo đảm GitOps luôn chạy chính xác 100% phiên bản container mong muốn.
*   **Bắt lỗi sớm trước khi Deploy:** Container Smoke Test giúp triệt tiêu hoàn toàn rủi ro container bị lỗi khởi động (CrashLoopBackOff) trên cụm K8s thật, đảm bảo chỉ có container hoạt động tốt mới được đẩy lên GHCR.
*   **Tối giản quy trình phát hành:** Loại bỏ hoàn toàn quy trình gắn tag release thủ công.

### Đánh đổi (Negatives):
*   **Thời gian chạy CI tăng nhẹ:** Do phải mất thêm thời gian để khởi chạy container thử nghiệm và kiểm tra sức khỏe. Tuy nhiên, việc đánh đổi này là hoàn toàn xứng đáng để có được sự an toàn tuyệt đối cho hệ thống GitOps.
