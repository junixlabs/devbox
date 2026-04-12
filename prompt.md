1. dùng mcp check xem có xác định được chưa
2. main là production, staging là staging, issue dùng format đó đi. Strategy dùng merge commit. đi đúng flow git nhưng chưa cần push, hãy vẫn cứ merge main nhưng test ở local. Có thể merge tất cả vào main trước rồi tách staging sau khi release public bản đầu tiên
3. khi dev xong cần tự review với simplify skill trước. Ở mỗi giai đoạn có task review code thì cần mở session mới và review với vai trò reviewer để đảm bảo code clean và merge chính xác. Giai đoạn này đang phát triển local first
4. dev 1 là device này, tôi đặt ngữ cảnh từ 1 máy mac khác nếu bạn cần tài nguyên test tôi sẽ cấp VPS mới
5. test ở device này, nếu cần vps thì tôi cấp. workflow phải chạy hoàn thiện
6. tôi cần hoàn thiện các phase ở mức tối đa thay vì dừng lại ở phase 0, đây là giai đoạn triển khai nên làm được cần nhiều càng tốt không có giới hạn