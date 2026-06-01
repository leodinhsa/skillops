# Security Review Guidelines

Manual security review focuses on identifying vulnerabilities that automated
tools might miss, such as logic flaws and authorization issues. Apply this
checklist on every review — not only when security is explicitly requested.

---

## 1. Authentication & Authorization

- **Role Validation**: Is the user's role checked before executing sensitive
  business logic?
- **Ownership**: Does the code verify that the user owns the resource they are
  trying to modify (e.g., an ownership ID check before mutation)?
- **Token Handling**: Are tokens or secrets ever logged or stored in plain text?

---

## 2. Data Validation & Injection

- **SQL Injection**: Even with ORMs, check for raw query execution or string
  formatting used to build query conditions.
- **Path Traversal**: Ensure file paths derived from user input are sanitized
  and cannot reference parent directories.
- **External Input**: Is input from APIs or users validated against a schema
  or type system before processing?
  *(See the language-specific reference file for the validation approach
  appropriate to your stack.)*

---

## 3. Resource Exhaustion

- **Unbounded Input**: Can a user send a massive payload that crashes or
  degrades the service (e.g., very large JSON bodies, deeply nested structures)?
- **Timeouts**: Do external API and service requests have explicit timeouts
  defined?
- **Pagination**: Are all list endpoints paginated to prevent unbounded
  result sets?

---

## 4. Cryptography

- **Secrets Generation**: Are security-sensitive tokens and identifiers
  generated using a cryptographically secure source, not a general-purpose
  random number generator?
  *(See the language-specific reference file for the correct module or
  library to use in your stack.)*
- **Password Hashing**: Are passwords hashed using a strong, slow algorithm
  (e.g., Argon2, BCrypt) before storage? Never MD5 or SHA-1 for passwords.

---

## 5. Exposure of Sensitive Information

- **Logging**: Is personally identifiable information (PII), passwords, or
  session tokens being written to logs?
- **Error Responses**: Do API errors leak stack traces, internal file paths,
  or environment variable values to the client?

---

> [!WARNING]
> If you find a security vulnerability, do NOT document the exploit in a
> public comment, PR description, or commit message. Document the risk and
> the fix pattern privately or via secure channels as required by your
> project's security disclosure process.