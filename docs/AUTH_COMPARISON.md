# Authentication Architecture Comparison
## Basic Auth + HttpOnly Cookie vs. OAuth 2.0 Authorization Code Flow (PKCE)

> Cả hai phương pháp đều giả định token được lưu trong **HttpOnly Cookie** — loại bỏ biến thể `localStorage` để so sánh thuần kiến trúc.

---

| Dimension | Basic Auth + HttpOnly Cookie | OAuth 2.0 Authorization Code + PKCE |
| :--- | :--- | :--- |
| **Authentication Flow** | Credential (username/password) được gửi trực tiếp đến Resource Server. Server xác thực và issue token trong cùng một request. | Tách biệt hai giai đoạn: Authentication (Gateway xác thực identity) và Authorization (Client nhận `code`, trao đổi lấy token). Credential không bao giờ rời khỏi Authorization Server. |
| **Token Issuance Surface** | Client Application trực tiếp nhận token từ Resource Server. Credential exposure nằm ở client layer. | Client chỉ nhận `authorization_code` (one-time, short-lived). Token được issue tại back-channel sau khi PKCE `code_verifier` được xác minh. Ngăn chặn **Authorization Code Interception Attack**. |
| **Credential Delegation (Third-Party)** | Third-party application buộc phải thu thập credential của người dùng để authenticate. Vi phạm nguyên tắc Zero-Trust. | Third-party application không bao giờ tiếp xúc credential. Người dùng authenticate trực tiếp với Authorization Server (Gateway). Token được cấp phát theo phạm vi (`scope`) được ủy quyền. |
| **Single Sign-On (SSO)** | Không có cơ chế native SSO. Mỗi application phải duy trì session độc lập. Horizontal scaling yêu cầu shared session store hoặc JWT stateless. | Native SSO thông qua SSO Session Cookie (`session_id`, 30 ngày) tại Authorization Server. Client Application thực hiện `GET /authorize` — nếu SSO session còn hạn, Gateway issue `code` và redirect ngay lập tức, zero user interaction. |
| **Cross-Domain Authorization** | Cookie bị giới hạn bởi Same-Origin Policy. Không thể chia sẻ session across multiple domains mà không có workaround (CORS, subdomain cookie, v.v.). | Không phụ thuộc vào cookie sharing. Authorization Code được truyền qua redirect URI — có thể cấp quyền cho bất kỳ domain nào đã đăng ký `redirect_uri` hợp lệ. |
| **Machine-to-Machine (M2M)** | Không có flow M2M chuẩn hóa. Thường được giải quyết bằng service account credential hoặc pre-shared API key — khó audit và rotate. | **Client Credentials Grant** (RFC 6749): Mỗi service được cấp `client_id` + `client_secret` riêng biệt. Token issue không cần user context. Role `SERVICE` được kiểm soát độc lập tại Gateway. |
| **Session Binding & Anti-Hijacking** | Session không bị ràng buộc với bất kỳ context nào ngoài cookie. Nếu cookie bị exfiltrate, attacker có thể replay từ bất kỳ origin nào. | `session_id` và `refresh_token` được bind với `SourceIP` tại thời điểm issue (embedded trong JWT payload). Mọi request với IP không khớp sẽ bị từ chối — ngăn chặn **Session Fixation** và **Token Replay Attack**. |
| **Authorization Code Lifecycle** | N/A | One-time use, short-lived (configurable `JWT_AUTH_CODE_EXP_MINUTES`). Code bị revoke ngay sau khi sử dụng hoặc hết hạn. Ngăn chặn **Replay Attack** trên authorization endpoint. |
| **Token Revocation Granularity** | Revocation thường là all-or-nothing (invalidate toàn bộ session). | Revocation có thể thực hiện theo từng `client_id`. Có thể revoke access của một application mà không ảnh hưởng đến các application khác của cùng user. |
| **Scalability & Ecosystem** | Phù hợp với single-application architecture. Mở rộng sang multi-application đòi hỏi refactor đáng kể. | Được thiết kế cho multi-tenant, multi-application ecosystem. Thêm client mới chỉ cần đăng ký `client_id` + `redirect_uri` — không thay đổi core auth logic. |
| **Standards Compliance** | Không theo chuẩn OAuth 2.0 / OIDC. Khó tích hợp với external Identity Provider (Google, Azure AD, Okta). | Tuân thủ RFC 6749 (OAuth 2.0), RFC 7636 (PKCE). Có thể mở rộng sang OIDC và federation với external IdP. |
| **Engineering Complexity** | Thấp. Authentication logic nằm gọn trong một endpoint. Phù hợp với MVP hoặc internal tooling. | Cao hơn ở Authorization Server layer (Client registry, PKCE verification, Authorization Code store, redirect URI validation). Trade-off hợp lý cho production-grade, multi-client system. |

---

## Summary

| | Basic Auth + Cookie | OAuth 2.0 PKCE Gateway |
|:---|:---:|:---:|
| **Use Case** | Single-app, MVP, internal tool | Multi-app, third-party, enterprise |
| **Security Baseline** | Adequate | Defense-in-depth |
| **SSO Support** | Manual | Native |
| **M2M Support** | Ad-hoc / workaround | RFC-compliant (`client_credentials`) |
| **Standards** | Custom | RFC 6749 + RFC 7636 + OIDC-ready |
| **Engineering Cost** | Low | High (justified at scale) |
