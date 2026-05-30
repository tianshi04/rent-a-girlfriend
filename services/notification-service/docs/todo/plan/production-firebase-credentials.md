# 🔒 KẾ HOẠCH BẢO MẬT CREDENTIALS FIREBASE TRÊN PRODUCTION (Production Firebase Credentials Plan)

Tài liệu này vạch ra kế hoạch lưu trữ và nạp thông tin xác thực của **Firebase Admin SDK** một cách an toàn trên môi trường Production (như Docker, Kubernetes, Cloud). Quyết định này giúp loại bỏ hoàn toàn rủi ro rò rỉ file khóa JSON bí mật lên các kho lưu trữ mã nguồn mở (Git).

---

## 1. Vấn đề Bảo mật & Nguyên tắc Vàng
*   **Vấn đề**: File JSON service account (Ví dụ: `notification-service-d166c-firebase-adminsdk-*.json`) chứa khóa riêng tư (Private Key) có toàn quyền truy cập hạ tầng Firebase.
*   **Nguyên tắc Vàng**: 
    1.  **Tuyệt đối KHÔNG bao giờ commit file JSON lên Git**. (Đã cấu hình chặn bằng `.gitignore`).
    2.  **Tuyệt đối KHÔNG đóng gói file JSON vào Docker Image** lúc build. (Vì Docker Image có thể bị đẩy lên Public Registry hoặc bị đảo ngược phân tích).

---

## 2. Các Phương án Triển khai An toàn trên Production

### Phương án A: Sử dụng Kubernetes Secrets & Volume Mount (Khuyến nghị cho Cluster)
Hạ tầng Kubernetes cung cấp cơ chế lưu trữ Secret chuyên biệt và mount động vào Pod tại thời điểm khởi chạy.

#### Bước 1: Tạo Kubernetes Secret từ file JSON thật
Mã hóa file JSON của bạn thành Kubernetes Secret (thực hiện trên máy quản trị hoặc luồng CI/CD bảo mật):
```bash
kubectl create secret generic firebase-admin-secret \
  --from-file=service-account.json=./notification-service-d166c-firebase-adminsdk-fbsvc-4b6bc988c1.json \
  -n rentagf-production
```

#### Bước 2: Cấu hình Kubernetes Deployment yaml để Mount File
Trong cấu hình Deploy Pod, mount Secret trên thành một file vật lý nằm ở một thư mục bảo mật trong container (Ví dụ: `/etc/secrets/firebase/`):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
spec:
  template:
    spec:
      containers:
      - name: notification-app
        image: rentagf/notification-service:latest
        volumeMounts:
        - name: firebase-volume
          mountPath: "/etc/secrets/firebase"
          readOnly: true
        env:
        # Cấu hình biến môi trường trỏ tới file mount
        - name: FIREBASE_CONFIG_PATH
          value: "/etc/secrets/firebase/service-account.json"
      volumes:
      - name: firebase-volume
        secret:
          secretName: firebase-admin-secret
```

#### Bước 3: Đọc file trong ứng dụng Spring Boot
Cập nhật file `application.yml` để đọc đường dẫn file từ biến môi trường:
```yaml
notification:
  firebase:
    config-path: ${FIREBASE_CONFIG_PATH:classpath:simulated-mode}
```
Code Java sẽ kiểm tra: Nếu `config-path` không phải mock/simulated, thực hiện tạo `FileInputStream` nạp cấu hình thật.

---

### Phương án B: Base64 Encoded Environment Variable (Dễ triển khai cho Docker/VPS)
Phù hợp cho môi trường chạy Docker Compose độc lập hoặc deploy trên các Cloud Provider đơn giản (Render, Heroku, AWS ECS).

#### Bước 1: Mã hóa Base64 nội dung file JSON
Mã hóa toàn bộ nội dung file JSON thành một chuỗi văn bản duy nhất (Base64 string):
```bash
# Trên Linux/macOS
cat notification-service-*.json | base64 -w 0

# Trên Windows (PowerShell)
[Convert]::ToBase64String([System.IO.File]::ReadAllBytes("notification-service-d166c-firebase-adminsdk-fbsvc-4b6bc988c1.json"))
```
*Kết quả sẽ trả về một chuỗi ký tự dài dạng: `ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIs...`*

#### Bước 2: Nạp chuỗi Base64 vào biến môi trường Production
Cấu hình biến môi trường `FIREBASE_CREDENTIALS_BASE64` cho container:
```env
FIREBASE_CREDENTIALS_BASE64=ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIs...
```

#### Bước 3: Xử lý giải mã trong Code Java
Trong cấu hình Startup, kiểm tra sự tồn tại của biến môi trường Base64 này để nạp trực tiếp qua Stream:
```java
String base64Credentials = System.getenv("FIREBASE_CREDENTIALS_BASE64");
if (base64Credentials != null && !base64Credentials.trim().isEmpty()) {
    byte[] decodedBytes = Base64.getDecoder().decode(base64Credentials.trim());
    try (InputStream serviceAccount = new ByteArrayInputStream(decodedBytes)) {
        FirebaseOptions options = FirebaseOptions.builder()
            .setCredentials(GoogleCredentials.fromStream(serviceAccount))
            .build();
        FirebaseApp.initializeApp(options);
        log.info("Firebase Admin SDK initialized successfully via Base64 env variable.");
    }
}
```

---

### Phương án C: GCP Workload Identity (Bảo mật Tuyệt đối, Không cần Key File)
Nếu dự án được deploy trên hạ tầng Google Cloud Platform (GCP) như **Google Kubernetes Engine (GKE)** hoặc **Cloud Run**, chúng ta hoàn toàn không cần tạo hay quản lý bất kỳ file JSON Service Account nào!

1.  **Workload Identity** liên kết Kubernetes Service Account (KSA) chạy Pod trực tiếp với Google Service Account (GSA) được cấp quyền Admin FCM.
2.  Trong code Java, ta chỉ cần gọi Application Default Credentials (ADC):
    ```java
    FirebaseOptions options = FirebaseOptions.builder()
        .setCredentials(GoogleCredentials.getApplicationDefault())
        .build();
    FirebaseApp.initializeApp(options);
    ```
3.  Google Cloud sẽ tự động trao đổi mã token bảo mật định kỳ ở tầng hạ tầng. Không có private key nào được tạo ra, loại bỏ 100% nguy cơ lộ lọt key file.
