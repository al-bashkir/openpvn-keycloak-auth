# Graph Report - .  (2026-05-02)

## Corpus Check
- 77 files · ~83,864 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 482 nodes · 1067 edges · 21 communities detected
- Extraction: 66% EXTRACTED · 34% INFERRED · 0% AMBIGUOUS · INFERRED: 359 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_IPC Protocol Examples|IPC Protocol Examples]]
- [[_COMMUNITY_IPC Protocol Types|IPC Protocol Types]]
- [[_COMMUNITY_IPC Server & Peer Credentials|IPC Server & Peer Credentials]]
- [[_COMMUNITY_Non-Linux Peercred Stub|Non-Linux Peercred Stub]]
- [[_COMMUNITY_Auth Script Handler|Auth Script Handler]]
- [[_COMMUNITY_Auth Control File Writers|Auth Control File Writers]]
- [[_COMMUNITY_Main Entrypoints & OIDC Provider|Main Entrypoints & OIDC Provider]]
- [[_COMMUNITY_Token Validator|Token Validator]]
- [[_COMMUNITY_OIDC Crypto Primitives|OIDC Crypto Primitives]]
- [[_COMMUNITY_HTTP Middleware & Rate Limiting|HTTP Middleware & Rate Limiting]]
- [[_COMMUNITY_Health Endpoint|Health Endpoint]]
- [[_COMMUNITY_HTTP Server & Session Tests|HTTP Server & Session Tests]]
- [[_COMMUNITY_Daemon Auth Request Handler|Daemon Auth Request Handler]]
- [[_COMMUNITY_Configuration & Validation|Configuration & Validation]]
- [[_COMMUNITY_Session Type|Session Type]]
- [[_COMMUNITY_Project Worklog & Decisions|Project Worklog & Decisions]]
- [[_COMMUNITY_OIDC Security Hardening|OIDC Security Hardening]]
- [[_COMMUNITY_HTML Templates|HTML Templates]]
- [[_COMMUNITY_Auth Flow & Control Files|Auth Flow & Control Files]]
- [[_COMMUNITY_Build & Hardening Infrastructure|Build & Hardening Infrastructure]]
- [[_COMMUNITY_Testing & CI|Testing & CI]]

## God Nodes (most connected - your core abstractions)
1. `NewServer() [httpserver]` - 28 edges
2. `README - OpenVPN Keycloak SSO` - 23 edges
3. `NewManager()` - 21 edges
4. `Manager` - 15 edges
5. `New()` - 14 edges
6. `handleAuthRequest()` - 14 edges
7. `Sanitize()` - 13 edges
8. `OpenVPN Keycloak SSO Project Worklog` - 12 edges
9. `PKCE (RFC 7636) with S256 challenge` - 12 edges
10. `NewServer() [ipc]` - 12 edges

## Surprising Connections (you probably didn't know these)
- `Attack surface analysis: URL interception scenarios` --semantically_similar_to--> `False positive: open-redirect via sess.AuthURL`  [INFERRED] [semantically similar]
  docs/attack.md → reports/SECURITY_REPORT.md
- `OIDC nonce binding ID token to flow` --semantically_similar_to--> `CSRF state parameter (16-byte crypto/rand)`  [INFERRED] [semantically similar]
  CHANGELOG.md → README.md
- `Log Sanitization (CWE-117)` --semantically_similar_to--> `Input Validation Regex Patterns`  [INFERRED] [semantically similar]
  docs/authflow.md → tasks/015-security-hardening.md
- `Decision: OIDC Authorization Code Flow with PKCE` --rationale_for--> `PKCE (RFC 7636) with S256 challenge`  [EXTRACTED]
  WORKLOG.md → README.md
- `Decision: OIDC Authorization Code Flow with PKCE` --rationale_for--> `Scenario: callback URL replay defeated by PKCE`  [INFERRED]
  WORKLOG.md → docs/attack.md

## Hyperedges (group relationships)
- **Authentication flow security stack (PKCE + state + nonce + JWKS)** — concept_pkce_s256, concept_csrf_state, concept_oidc_nonce_binding, concept_jwt_jwks_validation [EXTRACTED 0.95]
- **OpenVPN SSO deferred auth flow components** — concept_openvpn_deferred_auth, concept_web_auth_url, worklog_task005_auth_script_mode, worklog_task004_unix_socket_ipc, worklog_task010_authfile_writing [EXTRACTED 0.90]
- **Security audit artifacts (report + remediation + checklist)** — security_report_audit, remediation_patch_recommendations, security_checklist_doc, security_doc [INFERRED 0.85]
- **End-to-End Deferred Authentication Flow** — mode_auth, ipc_unix_socket, session_concept, pkce_flow_concept, auth_pending_file_concept, auth_control_file_concept, exit_code_2_deferred [EXTRACTED 0.95]
- **Fail-Secure Auth Result Guarantee** — fail_secure_principle, result_written_atomic, background_cleanup, writing_order_failure, auth_control_file_concept [EXTRACTED 0.90]
- **Client-Side SSO Browser Flow** — web_auth_url, iv_sso_capability, auth_retry_interact, tunnelblick_client, openvpn_connect_client, openvpn_cli_client [EXTRACTED 0.90]

## Communities

### Community 5 - "IPC Protocol Examples"
Cohesion: 0.18
Nodes (18): Client, NewClient(), TestClientServerCommunication(), TestServerHandlerError(), TestClientConnectionFailure(), TestServerSocketPermissions(), TestServerGracefulShutdown(), TestServerStopIsIdempotent() (+10 more)

### Community 15 - "IPC Protocol Types"
Cohesion: 0.6
Nodes (3): MessageType, AuthRequest, AuthResponse

### Community 11 - "IPC Server & Peer Credentials"
Cohesion: 0.21
Nodes (5): AuthRequestHandler, Server, validatePeerCredentials(), socketFileDescriptor(), currentEffectiveUID()

### Community 17 - "Non-Linux Peercred Stub"
Cohesion: 0.67
Nodes (1): validatePeerCredentials()

### Community 4 - "Auth Script Handler"
Cohesion: 0.11
Nodes (21): sanitizeIPCValue(), WriteAuthPending(), WriteAuthSuccess(), WriteAuthFailure(), Handler, NewHandler(), sanitizeValues(), readCredentialsFile() (+13 more)

### Community 6 - "Auth Control File Writers"
Cohesion: 0.15
Nodes (10): TestWriteAuthPending(), TestWriteAuthSuccess(), TestWriteAuthFailure(), TestWriteAuthSuccess_UnwritableDir(), TestWriteAuthFailure_UnwritableControlFile(), TestWriteOrder(), Server, safeOAuthErrorCode() (+2 more)

### Community 8 - "Main Entrypoints & OIDC Provider"
Cohesion: 0.16
Nodes (17): newTestIssuer(), TestNewProviderAndStartAuthFlow(), TestNewProvider_DiscoveryFailure(), NewProvider(), init(), main(), runServe(), runAuth() (+9 more)

### Community 9 - "Token Validator"
Cohesion: 0.25
Nodes (13): Validator, NewValidator(), getClaimString(), getRolesFromClaim(), getNestedClaim(), containsRole(), TestValidateToken_UsernameOnly(), TestValidateToken_WithRoles() (+5 more)

### Community 3 - "OIDC Crypto Primitives"
Cohesion: 0.14
Nodes (28): Provider, AuthFlowData, TokenData, mergeMissingClaim(), mergeMissingMapValues(), accessTokenClaimsErrorCategory(), accessTokenIssuedForClient(), decodeJWTPayload() (+20 more)

### Community 10 - "HTTP Middleware & Rate Limiting"
Cohesion: 0.24
Nodes (10): loggingMiddleware(), recoveryMiddleware(), ipEntry, IPRateLimiter, newIPRateLimiter(), rateLimitMiddleware(), extractIP(), redactRequestPath() (+2 more)

### Community 18 - "Health Endpoint"
Cohesion: 0.67
Nodes (1): HealthResponse

### Community 2 - "HTTP Server & Session Tests"
Cohesion: 0.14
Nodes (33): TestNewServer(), TestHealthEndpoint(), TestAuthRedirectEndpoint(), TestAuthRedirectEndpointWithBasePath(), TestCallbackEndpointWithBasePath(), TestCallbackEndpointValidParams(), TestCallbackEndpointMissingCode(), TestCallbackConcurrentSameState() (+25 more)

### Community 12 - "Daemon Auth Request Handler"
Cohesion: 0.33
Nodes (11): newTestOIDCIssuer(), TestBuildShortAuthURL(), TestValidateOpenVPNResultPath(), TestNewAndHandleAuthRequest_Success(), TestHandleAuthRequest_PendingWriteFailureWritesAuthFailure(), TestRun_HTTPServerStartFailureStopsAndReturnsError(), Daemon, New() (+3 more)

### Community 7 - "Configuration & Validation"
Cohesion: 0.17
Nodes (19): Config, ListenConfig, OIDCConfig, AuthConfig, TLSConfig, LogConfig, Load(), DefaultConfig() (+11 more)

### Community 19 - "Session Type"
Cohesion: 0.67
Nodes (1): Session

### Community 14 - "Project Worklog & Decisions"
Cohesion: 0.19
Nodes (13): OpenVPN Keycloak SSO Project Worklog, TASK-001 Project Setup, TASK-002 CLI Skeleton, TASK-003 Config Loading, TASK-004 Unix Socket IPC, TASK-005 Auth Script Mode, TASK-006 HTTP Server, TASK-007 OIDC Authorization Code Flow (+5 more)

### Community 1 - "OIDC Security Hardening"
Cohesion: 0.06
Nodes (51): TASK-009 Session Management, TASK-010 Auth Control File Writing, Decision: Script-based deferred auth (no C plugin), Decision: OIDC Authorization Code Flow with PKCE, README - OpenVPN Keycloak SSO, Architecture diagram (User -> OpenVPN -> Daemon -> Keycloak), Alternative: openvpn-auth-oauth2, Supported VPN Clients matrix (+43 more)

### Community 20 - "HTML Templates"
Cohesion: 1.0
Nodes (2): Authentication success HTML template, Authentication error HTML template

### Community 0 - "Auth Flow & Control Files"
Cohesion: 0.03
Nodes (84): OpenVPN Client Setup Guide, Client Compatibility Matrix, .ovpn Client Profile, Tunnelblick (macOS), OpenVPN Connect Client, Linux NetworkManager, OpenVPN CLI, WEB_AUTH:: URL Mechanism (+76 more)

### Community 13 - "Build & Hardening Infrastructure"
Cohesion: 0.15
Nodes (15): Phase 5: Short URL Redirect, Log Sanitization (CWE-117), Per-IP Rate Limit (10/s, burst 50), Security Headers (CSP, HSTS, X-Frame-Options), Task 014: systemd Packaging, Task 015: Security Hardening, openvpn-keycloak-sso.service, systemd Sandboxing Directives (+7 more)

### Community 16 - "Testing & CI"
Cohesion: 0.5
Nodes (4): Task 016: Testing, GitHub Actions CI Pipeline, Table-driven Tests, go test -race Concurrent Access Tests

## Knowledge Gaps
- **66 isolated node(s):** `TASK-001 Project Setup`, `TASK-003 Config Loading`, `TASK-014 systemd Packaging`, `TASK-015 Security Hardening`, `Decision: Script-based deferred auth (no C plugin)` (+61 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Non-Linux Peercred Stub`** (3 nodes): `peercred_other.go`, `validatePeerCredentials()`, `peercred_other.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Health Endpoint`** (3 nodes): `health.go`, `HealthResponse`, `health.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Session Type`** (3 nodes): `session.go`, `Session`, `session.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `HTML Templates`** (2 nodes): `Authentication success HTML template`, `Authentication error HTML template`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Daemon Auth Request Handler` to `HTTP Server & Session Tests`, `OIDC Crypto Primitives`, `IPC Protocol Examples`, `Configuration & Validation`, `Main Entrypoints & OIDC Provider`?**
  _High betweenness centrality (0.043) - this node is a cross-community bridge._
- **Why does `NewServer() [httpserver]` connect `IPC Protocol Examples` to `HTTP Server & Session Tests`, `OIDC Crypto Primitives`, `Auth Script Handler`, `Main Entrypoints & OIDC Provider`, `HTTP Middleware & Rate Limiting`, `Daemon Auth Request Handler`?**
  _High betweenness centrality (0.041) - this node is a cross-community bridge._
- **Why does `Sanitize()` connect `Auth Script Handler` to `Main Entrypoints & OIDC Provider`, `HTTP Middleware & Rate Limiting`, `Daemon Auth Request Handler`?**
  _High betweenness centrality (0.033) - this node is a cross-community bridge._
- **Are the 25 inferred relationships involving `NewServer() [httpserver]` (e.g. with `newTestIssuer()` and `TestNewProvider_DiscoveryFailure()`) actually correct?**
  _`NewServer() [httpserver]` has 25 INFERRED edges - model-reasoned connections that need verification._
- **Are the 19 inferred relationships involving `NewManager()` (e.g. with `TestAuthRedirectEndpoint()` and `TestAuthRedirectEndpointWithBasePath()`) actually correct?**
  _`NewManager()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **Are the 11 inferred relationships involving `New()` (e.g. with `NewServer() [httpserver]` and `NewProvider()`) actually correct?**
  _`New()` has 11 INFERRED edges - model-reasoned connections that need verification._
- **What connects `TASK-001 Project Setup`, `TASK-003 Config Loading`, `TASK-014 systemd Packaging` to the rest of the system?**
  _66 weakly-connected nodes found - possible documentation gaps or missing edges._