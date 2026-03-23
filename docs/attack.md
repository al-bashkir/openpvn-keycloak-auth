# Attack Surface Analysis: URL Interception Scenarios

## Scenario 1: Attacker captures the short URL

```
WEB_AUTH::https://vpn.example.com:9000/auth/a1b2c3d4e5f6...
```

**How it could leak:** Sniffing the OpenVPN control channel (unlikely -- it's TLS-encrypted), shoulder surfing, or reading logs.

**What happens if attacker opens it:**
- They get 302-redirected to Keycloak login page
- They must **authenticate as the correct user** in Keycloak (password + MFA)
- If they somehow do authenticate (e.g., they ARE the user on another device), the VPN connects for the original session -- **but the attacker's browser just showed a success page, they don't get a VPN tunnel themselves**
- The `auth_control_file` write grants access to the **original OpenVPN session**, not to the attacker's machine

**Risk: Low.** The URL is just a redirect to "please log in." Without Keycloak credentials, it's useless.

---

## Scenario 2: Attacker captures the full Keycloak auth URL

```
https://keycloak.example.com/realms/.../auth?client_id=openvpn&code_challenge=E9Mel...&state=a1b2c3...
```

**Same situation as Scenario 1** -- this URL just leads to a login page. The `code_challenge` is a **hash** of the PKCE verifier (S256). The attacker cannot reverse it. They still need Keycloak credentials.

**Risk: Low.**

---

## Scenario 3: Attacker captures the callback URL (the dangerous one)

```
https://vpn.example.com:9000/callback?code=AUTH_CODE&state=a1b2c3d4...
```

**How it could leak:** Browser history, HTTP referer header, proxy logs, browser extension, malware on user's machine.

**What happens if attacker replays it:**

The attacker has the `authorization_code`. But the daemon exchanges it with Keycloak using:

```
code=AUTH_CODE
code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk  <-- only daemon knows this
client_secret=SECRET                                          <-- only daemon knows this
```

**This is exactly what PKCE prevents.** Keycloak will reject a token exchange without the correct `code_verifier` that matches the `code_challenge` sent earlier. The attacker doesn't have the verifier (it's stored in the daemon's in-memory session, never exposed via any URL or HTTP response).

Additionally:
- Authorization codes are **single-use** -- if the daemon already exchanged it, replay fails
- Authorization codes are **short-lived** (typically 30-60 seconds in Keycloak)
- The `state` parameter is deleted from the session store after first use

**Risk: Very low** thanks to PKCE + single-use codes + client_secret.

---

## Scenario 4: Attacker is on the same network (MITM)

All URLs use **HTTPS**. To MITM:
- `vpn.example.com:9000` -- attacker needs a valid TLS cert for this domain
- `keycloak.example.com` -- attacker needs a valid TLS cert for this domain

Without compromising a CA, this is not feasible.

---

## Scenario 5: Full browser compromise

If the attacker has **full control of the user's browser** (malware, compromised extension), they could:
1. Wait for the user to authenticate in Keycloak
2. Intercept the callback **before** it reaches the daemon
3. Send it to the daemon themselves

But this gives them nothing extra -- the `auth_control_file` write connects the **original VPN session** (from the original user's machine). The attacker's machine doesn't get a tunnel.

If the attacker also controls the user's machine... they already have full access anyway.

---

## Summary

| Attack | Blocked by | Risk |
|--------|-----------|------|
| Steal short URL `/auth/<state>` | Keycloak login required | Low |
| Steal full Keycloak auth URL | Keycloak login required | Low |
| Steal callback `?code=...&state=...` | PKCE (verifier never exposed) + single-use code + client_secret | Very low |
| Replay callback after legitimate use | Single-use auth code + session deleted after first use | None |
| MITM any URL | TLS on all hops | Very low |
| Steal code + somehow get verifier | Verifier is in daemon memory only, never in any URL/response/log | Extremely low |

## Key Architectural Defense

**The secret (PKCE verifier) and the authorization code never travel through the same channel.** The code goes through the browser; the verifier stays in daemon memory. An attacker would need to compromise both the browser flow AND the daemon's memory simultaneously.
