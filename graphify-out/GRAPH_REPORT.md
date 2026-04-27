# Graph Report - openvpn-keycloak-auth  (2026-04-27)

## Corpus Check
- 37 files · ~82,468 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 260 nodes · 611 edges · 14 communities detected
- Extraction: 47% EXTRACTED · 53% INFERRED · 0% AMBIGUOUS · INFERRED: 325 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]

## God Nodes (most connected - your core abstractions)
1. `NewServer()` - 36 edges
2. `NewManager()` - 19 edges
3. `Manager` - 14 edges
4. `New()` - 13 edges
5. `handleAuthRequest()` - 13 edges
6. `Server` - 10 edges
7. `Sanitize()` - 10 edges
8. `NewClient()` - 9 edges
9. `Load()` - 9 edges
10. `TestDeleteSession()` - 9 edges

## Surprising Connections (you probably didn't know these)
- `NewServer()` --calls--> `TestRunAuth_Deferred()`  [INFERRED]
  internal/httpserver/server.go → cmd/openvpn-keycloak-auth/main_test.go
- `New()` --calls--> `runServe()`  [INFERRED]
  internal/daemon/daemon.go → cmd/openvpn-keycloak-auth/main.go
- `Load()` --calls--> `runServe()`  [INFERRED]
  internal/config/config.go → cmd/openvpn-keycloak-auth/main.go
- `Load()` --calls--> `runAuth()`  [INFERRED]
  internal/config/config.go → cmd/openvpn-keycloak-auth/main.go
- `Load()` --calls--> `runCheckConfig()`  [INFERRED]
  internal/config/config.go → cmd/openvpn-keycloak-auth/main.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.13
Nodes (25): NewClient(), ExampleClient(), ExampleServer(), authRoutes, TestCallbackEndpointOIDCError(), TestCallbackEndpointWithBasePath(), TestGracefulShutdown(), TestHealthEndpoint() (+17 more)

### Community 1 - "Community 1"
Cohesion: 0.12
Nodes (23): Handler, OpenVPNEnv, TestHandlerRun(), TestHandlerRunDaemonError(), TestHandlerRunNoDaemon(), TestParseEnv(), TestParseEnvWithIVSSO(), TestReadCredentialsFile() (+15 more)

### Community 2 - "Community 2"
Cohesion: 0.14
Nodes (20): accessTokenClaimsErrorCategory(), accessTokenIssuedForClient(), decodeJWTPayload(), generateCodeChallenge(), generateCodeVerifier(), generateState(), mergeMissingClaim(), mergeMissingMapValues() (+12 more)

### Community 3 - "Community 3"
Cohesion: 0.25
Nodes (19): TestAuthRedirectEndpoint(), TestAuthRedirectEndpointWithBasePath(), TestCallbackEndpointMissingCode(), generateSessionID(), NewManager(), TestClaimResultWrite(), TestCleanup(), TestConcurrentAccess() (+11 more)

### Community 4 - "Community 4"
Cohesion: 0.13
Nodes (16): SetupLogging(), TestSetupLogging(), getGoVersion(), runCheckConfig(), runServe(), runVersion(), TestRunAuth_Deferred(), TestRunCheckConfig_Invalid() (+8 more)

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (7): authStateFromPath(), safeOAuthErrorCode(), validationFailureCategory(), Daemon, Server, TestAuthStateFromPath(), Manager

### Community 6 - "Community 6"
Cohesion: 0.14
Nodes (17): AuthConfig, Config, DefaultConfig(), isReservedCallbackPath(), ListenConfig, Load(), LogConfig, OIDCConfig (+9 more)

### Community 7 - "Community 7"
Cohesion: 0.22
Nodes (13): Validator, containsRole(), getClaimString(), getNestedClaim(), getRolesFromClaim(), NewValidator(), TestContainsRole(), TestGetNestedClaim() (+5 more)

### Community 8 - "Community 8"
Cohesion: 0.18
Nodes (12): ipEntry, IPRateLimiter, TestExtractIP(), TestRedactRequestPath(), extractIP(), loggingMiddleware(), newIPRateLimiter(), rateLimitMiddleware() (+4 more)

### Community 9 - "Community 9"
Cohesion: 0.18
Nodes (6): AuthRequestHandler, Server, currentEffectiveUID(), socketFileDescriptor(), validatePeerCredentials(), sanitizeIPCValue()

### Community 10 - "Community 10"
Cohesion: 0.35
Nodes (10): buildShortAuthURL(), handleAuthRequest(), New(), newTestOIDCIssuer(), TestBuildShortAuthURL(), TestHandleAuthRequest_PendingWriteFailureWritesAuthFailure(), TestNewAndHandleAuthRequest_Success(), TestRun_HTTPServerStartFailureStopsAndReturnsError() (+2 more)

### Community 11 - "Community 11"
Cohesion: 0.5
Nodes (3): AuthRequest, AuthResponse, MessageType

### Community 13 - "Community 13"
Cohesion: 1.0
Nodes (1): HealthResponse

### Community 14 - "Community 14"
Cohesion: 1.0
Nodes (1): Session

## Knowledge Gaps
- **16 isolated node(s):** `MessageType`, `AuthRequest`, `AuthResponse`, `AuthRequestHandler`, `AuthFlowData` (+11 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 13`** (2 nodes): `HealthResponse`, `health.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 14`** (2 nodes): `session.go`, `Session`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewServer()` connect `Community 0` to `Community 1`, `Community 3`, `Community 4`, `Community 8`, `Community 9`, `Community 10`?**
  _High betweenness centrality (0.154) - this node is a cross-community bridge._
- **Why does `New()` connect `Community 10` to `Community 0`, `Community 2`, `Community 3`, `Community 4`?**
  _High betweenness centrality (0.116) - this node is a cross-community bridge._
- **Why does `Sanitize()` connect `Community 1` to `Community 8`, `Community 9`, `Community 10`, `Community 4`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **Are the 33 inferred relationships involving `NewServer()` (e.g. with `TestClientServerCommunication()` and `TestServerHandlerError()`) actually correct?**
  _`NewServer()` has 33 INFERRED edges - model-reasoned connections that need verification._
- **Are the 18 inferred relationships involving `NewManager()` (e.g. with `TestAuthRedirectEndpoint()` and `TestAuthRedirectEndpointWithBasePath()`) actually correct?**
  _`NewManager()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **Are the 11 inferred relationships involving `New()` (e.g. with `generateCodeChallenge()` and `TestGenerateCodeChallenge()`) actually correct?**
  _`New()` has 11 INFERRED edges - model-reasoned connections that need verification._
- **Are the 9 inferred relationships involving `handleAuthRequest()` (e.g. with `TestNewAndHandleAuthRequest_Success()` and `TestHandleAuthRequest_PendingWriteFailureWritesAuthFailure()`) actually correct?**
  _`handleAuthRequest()` has 9 INFERRED edges - model-reasoned connections that need verification._