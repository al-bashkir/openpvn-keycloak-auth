# Graph Report - .  (2026-05-13)

## Corpus Check
- 56 files · ~60,704 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 453 nodes · 833 edges · 38 communities (17 shown, 21 thin omitted)
- Extraction: 89% EXTRACTED · 11% INFERRED · 0% AMBIGUOUS · INFERRED: 92 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_HTTP Server & Callback Tests|HTTP Server & Callback Tests]]
- [[_COMMUNITY_Threat Model & Attack Surface|Threat Model & Attack Surface]]
- [[_COMMUNITY_OIDC PKCE Flow + Provider|OIDC PKCE Flow + Provider]]
- [[_COMMUNITY_IPC ClientServer Protocol|IPC Client/Server Protocol]]
- [[_COMMUNITY_Config Loading & Validation|Config Loading & Validation]]
- [[_COMMUNITY_Daemon Orchestration|Daemon Orchestration]]
- [[_COMMUNITY_Log Sanitization & Control Files|Log Sanitization & Control Files]]
- [[_COMMUNITY_HTTP Middleware & Rate Limiting|HTTP Middleware & Rate Limiting]]
- [[_COMMUNITY_Architecture Decisions|Architecture Decisions]]
- [[_COMMUNITY_Auth Script Mode Tests|Auth Script Mode Tests]]
- [[_COMMUNITY_CLI Entrypoint & Commands|CLI Entrypoint & Commands]]
- [[_COMMUNITY_OIDC Callback Handler|OIDC Callback Handler]]
- [[_COMMUNITY_OIDC Token Validator|OIDC Token Validator]]
- [[_COMMUNITY_SO_PEERCRED Peer Authentication|SO_PEERCRED Peer Authentication]]
- [[_COMMUNITY_IPC Protocol Types|IPC Protocol Types]]
- [[_COMMUNITY_Health Endpoint|Health Endpoint]]
- [[_COMMUNITY_Session Type|Session Type]]
- [[_COMMUNITY_File ipc_example_test_go|File: ipc_example_test_go]]
- [[_COMMUNITY_File ipc_ipc_test_go|File: ipc_ipc_test_go]]
- [[_COMMUNITY_File ipc_peercred_linux_go|File: ipc_peercred_linux_go]]
- [[_COMMUNITY_File ipc_peercred_other_go|File: ipc_peercred_other_go]]
- [[_COMMUNITY_File ipc_sanitize_go|File: ipc_sanitize_go]]
- [[_COMMUNITY_File logsanitize_sanitize_go|File: logsanitize_sanitize_go]]
- [[_COMMUNITY_File logsanitize_sanitize_test_go|File: logsanitize_sanitize_test_go]]
- [[_COMMUNITY_File oidc_flow_go|File: oidc_flow_go]]
- [[_COMMUNITY_File oidc_flow_test_go|File: oidc_flow_test_go]]
- [[_COMMUNITY_File oidc_provider_go|File: oidc_provider_go]]
- [[_COMMUNITY_File oidc_provider_test_go|File: oidc_provider_test_go]]
- [[_COMMUNITY_File oidc_validator_go|File: oidc_validator_go]]
- [[_COMMUNITY_File oidc_validator_test_go|File: oidc_validator_test_go]]
- [[_COMMUNITY_File openvpn_authfile_go|File: openvpn_authfile_go]]
- [[_COMMUNITY_File openvpn_authfile_test_go|File: openvpn_authfile_test_go]]
- [[_COMMUNITY_File session_cleanup_go|File: session_cleanup_go]]
- [[_COMMUNITY_File session_manager_go|File: session_manager_go]]
- [[_COMMUNITY_File session_session_go|File: session_session_go]]
- [[_COMMUNITY_File session_session_test_go|File: session_session_test_go]]

## God Nodes (most connected - your core abstractions)
1. `NewManager()` - 21 edges
2. `Security Guide (docs/security.md)` - 21 edges
3. `internal/httpserver/httpserver_test.go` - 16 edges
4. `Sanitize()` - 15 edges
5. `internal/config/config.go` - 15 edges
6. `Server.handleCallback` - 15 edges
7. `Architecture (docs/architecture.md)` - 15 edges
8. `New()` - 14 edges
9. `Manager` - 14 edges
10. `handleAuthRequest()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `Decision: Remove X-XSS-Protection Header` --describes--> `securityHeadersMiddleware()`  [INFERRED]
  CHANGELOG.md → internal/httpserver/middleware.go
- `Log Sanitization (CWE-117 mitigation, strips control chars)` --describes--> `Sanitize()`  [EXTRACTED]
  docs/authflow.md → internal/logsanitize/sanitize.go
- `Auth Flow Phase 9: Safety Nets (defer/cleanup/error paths)` --describes--> `Server.handleCallback`  [EXTRACTED]
  docs/authflow.md → internal/httpserver/callback.go
- `Claim Merging: Access Token JWT verified before merging realm_access/resource_access/groups` --describes--> `Provider.ExchangeCode`  [EXTRACTED]
  docs/authflow.md → internal/oidc/flow.go
- `Auth Flow Phase 2: Auth Script Execution` --describes--> `ParseEnv()`  [EXTRACTED]
  docs/authflow.md → internal/auth/envparser.go

## Hyperedges (group relationships)
- **OpenVPN auth deferred SSO request pipeline** — openvpn_keycloak_auth_main_runauth, auth_handler_run, ipc_client_sendauthrequest, ipc_server_handleconnection, daemon_daemon_handleauthrequest [INFERRED 0.90]
- **OIDC callback completion flow (token exchange to OpenVPN result)** — httpserver_callback_handlecallback, httpserver_callback_writeauthsuccess, httpserver_callback_writeauthfailure, httpserver_callback_safeoautherrorcode, httpserver_callback_validationfailurecategory [EXTRACTED 0.95]
- **HTTP middleware chain composing the server handler** — httpserver_server_newserver, httpserver_middleware_loggingmiddleware, httpserver_middleware_recoverymiddleware, httpserver_middleware_ratelimitmiddleware, httpserver_middleware_securityheadersmiddleware [EXTRACTED 0.95]
- **PKCE+Nonce OIDC Authorization Flow** — oidc_flow_startauthflow, oidc_flow_exchangecode, oidc_flow_generatecodeverifier, oidc_flow_generatecodechallenge, oidc_flow_generatestate, oidc_flow_generatenonce [EXTRACTED 1.00]
- **Session Result-Write Claim State Machine** — session_session_session, session_manager_claimresultwrite, session_manager_releaseresultwriteclaim, session_manager_markresultwritten, session_cleanup_cleanup [EXTRACTED 1.00]
- **OpenVPN Deferred Auth Control-File Trio** — openvpn_authfile_writeauthpending, openvpn_authfile_writeauthsuccess, openvpn_authfile_writeauthfailure, session_session_session [EXTRACTED 1.00]
- **End-to-End OIDC Authentication Flow (Phases 1-9)** — docs_authflow_phase1_connection_init, docs_authflow_phase2_auth_script_exec, docs_authflow_phase3_daemon_processes, docs_authflow_phase4_deferred_browser, docs_authflow_phase5_short_url_redirect, docs_authflow_phase6_keycloak_auth, docs_authflow_phase7_token_exchange, docs_authflow_phase8_result_written, docs_authflow_phase9_safety_nets [EXTRACTED 1.00]
- **PKCE + State + Nonce Defense Triad** — docs_security_pkce_s256, docs_security_csrf_state_param, docs_security_oidc_nonce_binding, docs_attack_scenario_callback_capture, docs_attack_key_arch_defense [EXTRACTED 0.95]
- **Unreleased Audit-Hardening Decisions (nonce, TLS 1.3, XSS removal, errors.Is)** — root_changelog_decision_oidc_nonce_binding, root_changelog_decision_tls13_min, root_changelog_decision_remove_xss_protection, root_changelog_decision_errors_is_serverclosed, root_changelog_removed_dead_helpers [EXTRACTED 1.00]

## Communities (38 total, 21 thin omitted)

### Community 0 - "HTTP Server & Callback Tests"
Cohesion: 0.07
Nodes (39): Session Lifecycle with Result-Write Claim, Auth Flow Phase 5: Short URL 302 Redirect, CSRF State Parameter, internal/httpserver/httpserver_test.go, TestAuthRedirectEndpoint(), TestAuthRedirectEndpointWithBasePath(), TestAuthStateFromPath(), TestCallbackConcurrentSameState() (+31 more)

### Community 1 - "Threat Model & Attack Surface"
Cohesion: 0.05
Nodes (49): Key Architectural Defense: Verifier and Code on Separate Channels, Attack Surface Analysis (docs/attack.md), Attack Scenario 5: Full Browser Compromise, Attack Scenario 3: Stolen Callback URL (PKCE-mitigated), Attack Scenario 2: Stolen Keycloak Auth URL (Low risk), Attack Scenario 4: MITM (TLS-mitigated), Attack Scenario 1: Stolen Short URL (Low risk), Auth Flow Phase 7: Token Exchange + Validation + Claim Merging (+41 more)

### Community 2 - "OIDC PKCE Flow + Provider"
Cohesion: 0.08
Nodes (39): PKCE OAuth2 Flow (RFC 7636), OIDC Nonce Binding (ID token to flow), AuthFlowData, exchangeCodeFixture, accessTokenClaimsErrorCategory(), accessTokenIssuedForClient(), Provider.ExchangeCode, generateCodeChallenge() (+31 more)

### Community 3 - "IPC Client/Server Protocol"
Cohesion: 0.09
Nodes (21): IPC Protocol (JSON over AF_UNIX), AuthRequestHandler, Client, ipc.Client struct, internal/ipc/client.go, NewClient(), ExampleClient(), ExampleServer() (+13 more)

### Community 4 - "Config Loading & Validation"
Cohesion: 0.13
Nodes (25): AuthConfig, Config, AuthConfig, Config struct, DefaultConfig(), internal/config/config.go, isReservedCallbackPath(), ListenConfig (+17 more)

### Community 5 - "Daemon Orchestration"
Cohesion: 0.13
Nodes (23): Daemon, buildShortAuthURL(), Daemon struct, internal/daemon/daemon.go, handleAuthRequest(), New(), internal/daemon/daemon_test.go, newTestOIDCIssuer() (+15 more)

### Community 6 - "Log Sanitization & Control Files"
Cohesion: 0.12
Nodes (19): CWE-117 Log Injection Defense, OpenVPN Deferred Auth Control File Protocol, Control File Write Order (reason before control), Auth Flow Phase 8: Result Written to OpenVPN, sanitizeIPCValue(), Sanitize(), TestSanitize(), TestSanitizeLargeInput() (+11 more)

### Community 7 - "HTTP Middleware & Rate Limiting"
Cohesion: 0.15
Nodes (19): authRoutes, internal/httpserver/health.go, Server.handleHealth, HealthResponse, ipEntry, IPRateLimiter, extractIP(), globalLimiter (package var) (+11 more)

### Community 8 - "Architecture Decisions"
Cohesion: 0.09
Nodes (25): Decision: In-Memory Sessions (not Redis), Decision: JSON Over Unix Socket (not Protocol Buffers), Decision: Script-Based Auth (not C plugin), Decision: Single Binary with 4 Modes, Decision: Unix Socket IPC (not HTTP), Four CLI Modes (serve/auth/version/check-config), Architecture (docs/architecture.md), Script-Based Deferred Authentication (OpenVPN 2.6 exit code 2) (+17 more)

### Community 9 - "Auth Script Mode Tests"
Cohesion: 0.17
Nodes (20): internal/auth/auth_test.go, TestHandlerRun(), TestHandlerRunDaemonError(), TestHandlerRunNoDaemon(), TestParseEnv(), TestParseEnvWithIVSSO(), TestReadCredentialsFile(), TestSelectPendingMethod() (+12 more)

### Community 10 - "CLI Entrypoint & Commands"
Cohesion: 0.17
Nodes (20): SetupLogging(), authCmd, checkConfigCmd, ExitError/ExitDeferred/ExitConfig, cmd/openvpn-keycloak-auth/main.go, main(), rootCmd (cobra root command), runAuth() (+12 more)

### Community 11 - "OIDC Callback Handler"
Cohesion: 0.16
Nodes (14): authStateFromPath(), internal/httpserver/callback.go, Server.handleAuthRedirect, Server.handleCallback, safeOAuthErrorCode(), validationFailureCategory(), Server.writeAuthFailure, Server.writeAuthSuccess (+6 more)

### Community 12 - "OIDC Token Validator"
Cohesion: 0.15
Nodes (18): OIDC Nonce Binding (id_token to flow), JWT Claim Validation (iss/aud/exp/iat/nbf), JWT Signature Verification via JWKS, Validator, containsRole(), getClaimString(), getNestedClaim(), getRolesFromClaim() (+10 more)

### Community 13 - "SO_PEERCRED Peer Authentication"
Cohesion: 0.38
Nodes (5): SO_PEERCRED Peer UID Authentication, currentEffectiveUID(), socketFileDescriptor(), validatePeerCredentials(), validatePeerCredentials()

### Community 14 - "IPC Protocol Types"
Cohesion: 0.5
Nodes (3): AuthRequest, AuthResponse, MessageType

## Knowledge Gaps
- **95 isolated node(s):** `OpenVPNEnv`, `ListenConfig`, `OIDCConfig`, `AuthConfig`, `TLSConfig` (+90 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **21 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Daemon Orchestration` to `HTTP Server & Callback Tests`, `OIDC PKCE Flow + Provider`, `IPC Client/Server Protocol`, `HTTP Middleware & Rate Limiting`, `CLI Entrypoint & Commands`?**
  _High betweenness centrality (0.215) - this node is a cross-community bridge._
- **Why does `Security Guide (docs/security.md)` connect `Threat Model & Attack Surface` to `HTTP Server & Callback Tests`, `OIDC PKCE Flow + Provider`, `Daemon Orchestration`, `Architecture Decisions`, `OIDC Token Validator`?**
  _High betweenness centrality (0.110) - this node is a cross-community bridge._
- **Why does `NewServer()` connect `HTTP Middleware & Rate Limiting` to `HTTP Server & Callback Tests`, `OIDC Callback Handler`, `Daemon Orchestration`?**
  _High betweenness centrality (0.100) - this node is a cross-community bridge._
- **Are the 17 inferred relationships involving `NewManager()` (e.g. with `New()` and `TestAuthRedirectEndpoint()`) actually correct?**
  _`NewManager()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **Are the 6 inferred relationships involving `Sanitize()` (e.g. with `.Run()` and `sanitizeValues()`) actually correct?**
  _`Sanitize()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `OpenVPNEnv`, `ListenConfig`, `OIDCConfig` to the rest of the system?**
  _95 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `HTTP Server & Callback Tests` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._