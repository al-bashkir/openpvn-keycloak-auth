# Graph Report - openvpn-keycloak-auth  (2026-04-26)

## Corpus Check
- 37 files · ~82,384 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 258 nodes · 607 edges · 13 communities detected
- Extraction: 46% EXTRACTED · 54% INFERRED · 0% AMBIGUOUS · INFERRED: 325 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]

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
- `NewHandler()` --calls--> `runAuth()`  [INFERRED]
  internal/auth/handler.go → cmd/openvpn-keycloak-auth/main.go
- `Load()` --calls--> `runAuth()`  [INFERRED]
  internal/config/config.go → cmd/openvpn-keycloak-auth/main.go
- `Load()` --calls--> `runCheckConfig()`  [INFERRED]
  internal/config/config.go → cmd/openvpn-keycloak-auth/main.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.12
Nodes (26): NewClient(), Daemon, ExampleClient(), ExampleServer(), authRoutes, TestCallbackEndpointOIDCError(), TestCallbackEndpointWithBasePath(), TestGracefulShutdown() (+18 more)

### Community 1 - "Community 1"
Cohesion: 0.11
Nodes (17): authStateFromPath(), safeOAuthErrorCode(), validationFailureCategory(), ipEntry, IPRateLimiter, Server, TestAuthStateFromPath(), TestExtractIP() (+9 more)

### Community 2 - "Community 2"
Cohesion: 0.21
Nodes (20): TestAuthRedirectEndpoint(), TestAuthRedirectEndpointWithBasePath(), TestCallbackEndpointMissingCode(), generateSessionID(), NewManager(), Manager, TestClaimResultWrite(), TestCleanup() (+12 more)

### Community 3 - "Community 3"
Cohesion: 0.12
Nodes (22): Handler, OpenVPNEnv, TestHandlerRun(), TestHandlerRunDaemonError(), TestHandlerRunNoDaemon(), TestParseEnv(), TestParseEnvWithIVSSO(), TestReadCredentialsFile() (+14 more)

### Community 4 - "Community 4"
Cohesion: 0.13
Nodes (20): AuthConfig, Config, DefaultConfig(), isReservedCallbackPath(), ListenConfig, Load(), LogConfig, OIDCConfig (+12 more)

### Community 5 - "Community 5"
Cohesion: 0.14
Nodes (18): accessTokenClaimsErrorCategory(), accessTokenIssuedForClient(), decodeJWTPayload(), generateCodeVerifier(), generateState(), mergeMissingClaim(), mergeMissingMapValues(), makeSignedAccessToken() (+10 more)

### Community 6 - "Community 6"
Cohesion: 0.16
Nodes (14): getGoVersion(), runAuth(), runCheckConfig(), runVersion(), TestRunAuth_Deferred(), TestRunCheckConfig_Invalid(), TestRunCheckConfig_Valid(), TestRunServe_ConfigLoadFailure() (+6 more)

### Community 7 - "Community 7"
Cohesion: 0.22
Nodes (13): Validator, containsRole(), getClaimString(), getNestedClaim(), getRolesFromClaim(), NewValidator(), TestContainsRole(), TestGetNestedClaim() (+5 more)

### Community 8 - "Community 8"
Cohesion: 0.2
Nodes (4): AuthRequestHandler, Server, validatePeerCredentials(), sanitizeIPCValue()

### Community 9 - "Community 9"
Cohesion: 0.29
Nodes (12): buildShortAuthURL(), handleAuthRequest(), New(), newTestOIDCIssuer(), TestBuildShortAuthURL(), TestHandleAuthRequest_PendingWriteFailureWritesAuthFailure(), TestNewAndHandleAuthRequest_Success(), TestRun_HTTPServerStartFailureStopsAndReturnsError() (+4 more)

### Community 10 - "Community 10"
Cohesion: 0.5
Nodes (3): AuthRequest, AuthResponse, MessageType

### Community 12 - "Community 12"
Cohesion: 1.0
Nodes (1): HealthResponse

### Community 13 - "Community 13"
Cohesion: 1.0
Nodes (1): Session

## Knowledge Gaps
- **16 isolated node(s):** `MessageType`, `AuthRequest`, `AuthResponse`, `AuthRequestHandler`, `AuthFlowData` (+11 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 12`** (2 nodes): `HealthResponse`, `health.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 13`** (2 nodes): `session.go`, `Session`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewServer()` connect `Community 0` to `Community 1`, `Community 2`, `Community 3`, `Community 6`, `Community 8`, `Community 9`?**
  _High betweenness centrality (0.155) - this node is a cross-community bridge._
- **Why does `New()` connect `Community 9` to `Community 0`, `Community 2`, `Community 4`, `Community 6`?**
  _High betweenness centrality (0.118) - this node is a cross-community bridge._
- **Why does `Sanitize()` connect `Community 3` to `Community 8`, `Community 1`, `Community 9`, `Community 6`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **Are the 33 inferred relationships involving `NewServer()` (e.g. with `TestClientServerCommunication()` and `TestServerHandlerError()`) actually correct?**
  _`NewServer()` has 33 INFERRED edges - model-reasoned connections that need verification._
- **Are the 18 inferred relationships involving `NewManager()` (e.g. with `TestAuthRedirectEndpoint()` and `TestAuthRedirectEndpointWithBasePath()`) actually correct?**
  _`NewManager()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **Are the 11 inferred relationships involving `New()` (e.g. with `generateCodeChallenge()` and `TestGenerateCodeChallenge()`) actually correct?**
  _`New()` has 11 INFERRED edges - model-reasoned connections that need verification._
- **Are the 9 inferred relationships involving `handleAuthRequest()` (e.g. with `TestNewAndHandleAuthRequest_Success()` and `TestHandleAuthRequest_PendingWriteFailureWritesAuthFailure()`) actually correct?**
  _`handleAuthRequest()` has 9 INFERRED edges - model-reasoned connections that need verification._