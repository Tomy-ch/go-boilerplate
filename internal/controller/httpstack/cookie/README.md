# cookie

English | [日本語](README.ja.md)

Middleware to enforce secure Cookie policies (Secure / HttpOnly / SameSite / Path / Domain / Max-Age).

## Public API

|Function / Type|Description|
|---|---|
|`NewSecurityCookie(p)`|Create `SecurityCookie` config from `SecureCookieConfig`|
|`Middleware(cfg)`|Return Echo middleware that rewrites Set-Cookie headers|
|`RewriteSetCookie(raw)`|Rewrite a raw Set-Cookie header string based on security policy|
|`SecurityCookie`|Configuration struct for cookie attribute enforcement|
