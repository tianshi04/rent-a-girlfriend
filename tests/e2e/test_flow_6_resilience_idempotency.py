import sys
import os
import time
import subprocess
import json

BASE_URL = "http://localhost:8000"
TEST_SAGA_ID = "resilience-saga-101"
TEST_EVENT_ID = "e0000000-0000-0000-0000-000000000099"
TEST_BOOKING_ID = "b0000000-0000-0000-0000-000000000002"
TEST_DISPUTE_ID = "e0000000-0000-0000-0000-000000000002"

def run_sql(db, sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "postgres", "-d", db, "-t", "-c", sql]
    res = subprocess.run(cmd, capture_output=True, text=True, encoding='utf-8')
    if res.returncode != 0:
        print(f"❌ SQL Error in {db}: {res.stderr}")
        sys.exit(1)
    return res.stdout.strip()

def run_python_in_dispute_container(code):
    cmd = ["docker", "exec", "-i", "dispute-service", "python"]
    res = subprocess.run(cmd, input=code, capture_output=True, text=True, encoding='utf-8')
    if res.returncode != 0:
        print(f"❌ Container execution error: {res.stderr}")
        return None
    return res.stdout.strip()

def main():
    print("=============================================================")
    print("   LUỒNG 6: KIỂM THỬ TÍNH CHỊU LỖI & TÍNH TRÙNG LẶP (E2E)    ")
    print("=============================================================\n")

    # -------------------------------------------------------------------------
    # PHẦN 1: KIỂM THỬ TÍNH CHỊU LỖI (SAGA CRASH & RESUME - FAULT TOLERANCE)
    # -------------------------------------------------------------------------
    print("👉 [PHẦN 1] Kiểm thử tính chịu lỗi (Fault Tolerance) của SAGA...")
    
    # 1.1 Reset trạng thái ban đầu của dữ liệu
    print("-> [Setup 1] Reset các bản ghi interaction_service & booking_db về trạng thái trước xử lý...")
    run_sql("interaction_service", f"UPDATE reviews SET is_visible = true WHERE booking_id = '{TEST_BOOKING_ID}';")
    run_sql("interaction_service", f"UPDATE chat_rooms SET status = 'ACTIVE' WHERE booking_id = '{TEST_BOOKING_ID}';")
    run_sql("booking_db", f"UPDATE bookings SET status = 'DISPUTED' WHERE id = '{TEST_BOOKING_ID}';")
    run_sql("dispute_service", f"DELETE FROM saga_states WHERE saga_id = '{TEST_SAGA_ID}';")
    
    # Xác nhận dữ liệu ban đầu
    rev_vis = run_sql("interaction_service", f"SELECT is_visible FROM reviews WHERE booking_id = '{TEST_BOOKING_ID}';")
    room_stat = run_sql("interaction_service", f"SELECT status FROM chat_rooms WHERE booking_id = '{TEST_BOOKING_ID}';")
    book_stat = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{TEST_BOOKING_ID}';")
    print(f"   * Trạng thái review: {'Hiển thị' if rev_vis == 't' else 'Ẩn'} (Mong muốn: Hiển thị)")
    print(f"   * Trạng thái chat room: {room_stat} (Mong muốn: ACTIVE)")
    print(f"   * Trạng thái booking: {book_stat} (Mong muốn: DISPUTED)")

    # 1.2 Insert một saga_state đại diện cho server bị sập khi đang xử lý bước HIDING_REVIEW
    print("-> [Setup 2] Giả lập sập server bằng cách insert trực tiếp saga_state ở trạng thái lỗi...")
    insert_sql = f"""
    INSERT INTO saga_states (saga_id, dispute_id, booking_id, saga_type, current_state, retry_count, last_error, version, created_at, updated_at)
    VALUES ('{TEST_SAGA_ID}', '{TEST_DISPUTE_ID}', '{TEST_BOOKING_ID}', 'REFUND', 'HIDING_REVIEW', 0, 'Simulated connection crash on step 2', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
    """
    run_sql("dispute_service", insert_sql)
    print("   * Đã insert trạng thái SAGA 'HIDING_REVIEW' với lỗi 'Simulated connection crash'.")

    # 1.3 Chờ và giám sát tiến trình SagaRetryWorker chạy nền tự phục hồi
    print("-> Chờ SagaRetryWorker quét qua database để tự động chạy tiếp...")
    saga_resolved = False
    for i in range(12):
        time.sleep(1.0)
        saga_db = run_sql("dispute_service", f"SELECT current_state, retry_count, last_error FROM saga_states WHERE saga_id = '{TEST_SAGA_ID}';")
        if not saga_db:
            print("   * Không tìm thấy SAGA state.")
            continue
        parts = [p.strip() for p in saga_db.split('|')]
        curr_state, retries, err = parts[0], parts[1], parts[2]
        
        print(f"   * [Giây {i+1}] SAGA State: {curr_state} (Lần thử lại: {retries})")
        if curr_state == "DISPUTE_RESOLVED_REFUNDED":
            saga_resolved = True
            break

    if not saga_resolved:
        print("❌ Lỗi: SagaRetryWorker không tự phục hồi SAGA thành công sau 12 giây.")
        sys.exit(1)
        
    print("-> [SUCCESS] SagaRetryWorker đã tự động quét và hoàn thành SAGA thành công!")

    # 1.4 Xác thực kết quả phục hồi trong database các service khác
    print("-> [Xác thực 1] Kiểm tra tác vụ ẩn Review & khóa Chat Room trong interaction_service...")
    final_rev_vis = run_sql("interaction_service", f"SELECT is_visible FROM reviews WHERE booking_id = '{TEST_BOOKING_ID}';")
    final_room_stat = run_sql("interaction_service", f"SELECT status FROM chat_rooms WHERE booking_id = '{TEST_BOOKING_ID}';")
    
    if final_rev_vis == "f" and final_room_stat == "LOCKED":
        print("   -> [DATABASE SUCCESS] Đánh giá đã bị ẩn (is_visible=false) và Chat Room đã bị khóa (LOCKED)!")
    else:
        print(f"❌ [DATABASE ERROR] Sai lệch kết quả phục hồi: Review visible={final_rev_vis}, Room status={final_room_stat}")
        sys.exit(1)

    print("-> [Xác thực 2] Kiểm tra trạng thái Booking vẫn giữ nguyên là DISPUTED...")
    # SAGA retry chỉ thực thi các tích hợp hạ tầng còn thiếu (ẩn review/khóa chat)
    # chứ không tái xuất sự kiện DisputeResolved sang Booking Service (đã xuất trước đó khi Admin quyết định)
    final_book_stat = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{TEST_BOOKING_ID}';")
    if final_book_stat == "DISPUTED":
        print("   -> [DATABASE SUCCESS] Booking 2 giữ nguyên trạng thái DISPUTED chính xác!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking trong DB là '{final_book_stat}', mong đợi 'DISPUTED'")
        sys.exit(1)


    # -------------------------------------------------------------------------
    # PHẦN 2: KIỂM THỬ TÍNH TRÙNG LẶP (IDEMPOTENCY - GỬI 1 EVENT 2 LẦN)
    # -------------------------------------------------------------------------
    print("\n👉 [PHẦN 2] Kiểm thử tính trùng lặp (Idempotency) khi gửi 1 event 2 lần...")
    
    # 2.1 Reset dữ liệu kiểm thử idempotency
    print("-> [Setup] Reset trạng thái Booking 2 về ACCEPTED và xóa bản ghi sự kiện cũ...")
    run_sql("booking_db", f"UPDATE bookings SET status = 'ACCEPTED' WHERE id = '{TEST_BOOKING_ID}';")
    run_sql("booking_db", f"DELETE FROM processed_events WHERE event_id = '{TEST_EVENT_ID}';")
    
    cnt_before = int(run_sql("booking_db", f"SELECT COUNT(*) FROM processed_events WHERE event_id = '{TEST_EVENT_ID}';"))
    booking_stat_before = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{TEST_BOOKING_ID}';")
    print(f"   * Số lượng bản ghi sự kiện '{TEST_EVENT_ID}': {cnt_before} (Mong muốn: 0)")
    print(f"   * Trạng thái Booking 2: {booking_stat_before} (Mong muốn: ACCEPTED)")

    # 2.2 Viết script để gửi CloudEvent qua Kafka
    publish_code = f"""
import asyncio
import json
from aiokafka import AIOKafkaProducer

async def main():
    producer = AIOKafkaProducer(
        bootstrap_servers='kafka:9092',
        value_serializer=lambda v: json.dumps(v).encode('utf-8')
    )
    await producer.start()
    try:
        ce_dict = {{
            "specversion": "1.0",
            "id": "{TEST_EVENT_ID}",
            "source": "/rent-a-gf/dispute-service/resilience-test",
            "type": "dispute.dispute-created.v1",
            "datacontenttype": "application/json",
            "time": "2026-06-21T10:30:00Z",
            "data": {{
                "disputeId": "{TEST_DISPUTE_ID}",
                "bookingId": "{TEST_BOOKING_ID}",
                "reporterId": "c0000000-0000-0000-0000-000000000001",
                "reason": "MISCONDUCT"
            }}
        }}
        await producer.send_and_wait(
            topic="dispute.events",
            key=b"{TEST_BOOKING_ID}",
            value=ce_dict
        )
        print("PUBLISHED")
    except Exception as e:
        print(f"ERROR: {{e}}")
    finally:
        await producer.stop()

asyncio.run(main())
"""

    # 2.3 Gửi event lần thứ nhất
    print("-> Gửi event 'dispute.dispute-created.v1' lần thứ 1 vào Kafka...")
    out1 = run_python_in_dispute_container(publish_code)
    if "PUBLISHED" not in out1:
        print(f"❌ Lỗi khi gửi event lần 1: {out1}")
        sys.exit(1)
    print("   * [KAFKA SUCCESS] Đã publish event lần 1.")

    # Chờ consumer xử lý event
    print("-> Chờ booking-service xử lý sự kiện...")
    booking_stat_after_1 = ""
    for _ in range(10):
        booking_stat_after_1 = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{TEST_BOOKING_ID}';")
        if booking_stat_after_1 == "DISPUTED":
            break
        time.sleep(0.5)

    if booking_stat_after_1 == "DISPUTED":
        print("   -> [DATABASE SUCCESS] Booking 2 đã chuyển trạng thái sang DISPUTED thành công!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking sau event 1 là '{booking_stat_after_1}', mong đợi 'DISPUTED'")
        sys.exit(1)

    # Kiểm tra bảng processed_events
    cnt_after_1 = int(run_sql("booking_db", f"SELECT COUNT(*) FROM processed_events WHERE event_id = '{TEST_EVENT_ID}';"))
    if cnt_after_1 == 1:
        print(f"   -> [DATABASE SUCCESS] Đã lưu 1 bản ghi sự kiện đã xử lý với ID {TEST_EVENT_ID}")
    else:
        print(f"❌ [DATABASE ERROR] Số lượng bản ghi sự kiện trong DB là {cnt_after_1}, mong đợi 1")
        sys.exit(1)

    # 2.4 Gửi event trùng lặp lần thứ hai
    print("-> Gửi event 'dispute.dispute-created.v1' trùng lặp lần thứ 2...")
    out2 = run_python_in_dispute_container(publish_code)
    if "PUBLISHED" not in out2:
        print(f"❌ Lỗi khi gửi event lần 2: {out2}")
        sys.exit(1)
    print("   * [KAFKA SUCCESS] Đã publish event lần 2 (trùng lặp ID).")

    # Chờ một lúc để đảm bảo consumer đã xử lý
    print("-> Chờ 2 giây để consumer xử lý trùng lặp...")
    time.sleep(2.0)

    # Xác nhận số lượng processed_events và trạng thái booking vẫn không đổi
    booking_stat_after_2 = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{TEST_BOOKING_ID}';")
    cnt_after_2 = int(run_sql("booking_db", f"SELECT COUNT(*) FROM processed_events WHERE event_id = '{TEST_EVENT_ID}';"))
    
    print(f"   * Trạng thái Booking sau event 2: {booking_stat_after_2} (Mong muốn: DISPUTED)")
    print(f"   * Số lượng bản ghi sự kiện '{TEST_EVENT_ID}' sau event 2: {cnt_after_2} (Mong muốn: 1)")

    if cnt_after_2 == 1 and booking_stat_after_2 == "DISPUTED":
        print("-> [SUCCESS] Kiểm tra Idempotency thành công! Event trùng lặp bị bỏ qua an toàn và không xử lý lại.")
    else:
        print("❌ [DATABASE ERROR] Lỗi Idempotency: Bản ghi xử lý bị ghi đè hoặc booking bị xử lý trùng lặp.")
        sys.exit(1)

    print("\n=============================================================")
    print("   [SUCCESS] LUỒNG 6 HOÀN THÀNH XÁC MINH RESILIENCE & IDEMPOTENCY! ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
