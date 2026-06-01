# Java — Language-Specific Review Checklist

Load this file after `review-checklist.md` and `security-review.md` when
reviewing Java code. This file covers Java-specific checks that extend the
common checklist. For Spring Boot projects, also load `lang-java-spring.md`.

---

## Logic & Correctness — Java

- [ ] **`equals()` vs `==`**: Is `==` used to compare `String`, `Integer`,
  or other object types where `.equals()` is required? `==` checks reference
  identity, not value equality.

  ```java
  // Correct
  if (status.equals("ACTIVE")) { ... }

  // Wrong — may pass in tests but fail in production
  if (status == "ACTIVE") { ... }
  ```

- [ ] **`Optional` used safely**: Is `Optional.get()` called without a prior
  `.isPresent()` check or without using `.orElse()` / `.orElseThrow()`?
  A bare `.get()` on an empty Optional throws `NoSuchElementException`.

  ```java
  // Correct
  String name = optional.orElseThrow(
      () -> new ResourceNotFoundException("User not found")
  );

  // Wrong — throws NoSuchElementException with no context
  String name = optional.get();
  ```

- [ ] **`Collections` not modified during iteration**: Is a `List` or `Map`
  being modified while iterating over it with a for-each loop? This throws
  `ConcurrentModificationException` at runtime. Use an `Iterator` with
  `.remove()` or collect to a new list.
- [ ] **Integer overflow**: Is arithmetic performed on user-supplied `int`
  values that could overflow? Use `Math.addExact()` / `Math.multiplyExact()`
  or promote to `long` where overflow is a risk.
- [ ] **`NullPointerException` paths**: Are method calls chained on objects
  that could be null without a null check or `Optional` wrapper?

---

## Architecture & Design — Java

- [ ] **Immutability where possible**: Are fields declared `final` where
  they are set in the constructor and never reassigned? Immutable objects
  are inherently thread-safe and easier to reason about.
- [ ] **Interface over implementation**: Are dependencies typed against
  interfaces (`List`, `Map`, `UserRepository`) rather than concrete
  implementations (`ArrayList`, `HashMap`, `UserRepositoryImpl`)?
- [ ] **Access modifiers minimal**: Are classes, methods, and fields given
  the least permissive access modifier that still allows correct behaviour?
  `public` should be the exception, not the default.
- [ ] **Checked vs unchecked exceptions**: Is a checked exception used for
  a condition the caller can reasonably recover from? Is an unchecked
  exception used for a programming error that cannot be meaningfully
  handled by the caller?
- [ ] **`static` methods appropriate**: Is a `static` utility method
  mutating shared state, or does it have side effects? Static methods
  should be pure functions or stateless helpers.

---

## Readability & Maintainability — Java

- [ ] **Javadoc on public API**: Are all public classes, interfaces, and
  methods documented with Javadoc? The `@param`, `@return`, and `@throws`
  tags must be present and accurate.

  ```java
  /**
   * Transfers funds between two accounts.
   *
   * @param sourceId  the ID of the account to debit
   * @param targetId  the ID of the account to credit
   * @param amount    the amount to transfer, must be positive
   * @throws InsufficientFundsException if the source account balance
   *                                    is less than {@code amount}
   */
  public void transfer(long sourceId, long targetId, BigDecimal amount)
      throws InsufficientFundsException { ... }
  ```

- [ ] **Magic numbers named**: Are numeric literals used inline in logic?
  Extract them as `private static final` constants with a descriptive name.

  ```java
  // Correct
  private static final int SESSION_EXPIRY_SECONDS = 86_400;

  // Wrong
  if (elapsed > 86400) { ... }
  ```

- [ ] **`var` used judiciously**: Is `var` used where the inferred type is
  not immediately obvious from the right-hand side? `var` is appropriate
  when the type is explicit in the initialiser; it reduces clarity when
  the type must be inferred mentally.
- [ ] **Method length**: Does any method exceed the project threshold
  (default: 50 lines)? Long methods typically violate Single Responsibility
  and should be decomposed.

---

## Error Handling — Java

- [ ] **Specific exceptions caught**: Is `catch (Exception e)` or
  `catch (Throwable t)` used where a more specific type is available?
  Broad catches hide unexpected failures and make debugging harder.
- [ ] **`finally` or `try-with-resources`**: Are resources (streams,
  connections, readers) closed in a `finally` block or, preferably, via
  `try-with-resources`? A resource opened outside `try-with-resources`
  is not guaranteed to close on exception.

  ```java
  // Correct — always closed
  try (InputStream is = Files.newInputStream(path)) {
      process(is);
  }

  // Wrong — not closed if process() throws
  InputStream is = Files.newInputStream(path);
  process(is);
  is.close();
  ```

- [ ] **Exception wrapping preserves cause**: When catching and re-throwing
  a different exception type, is the original exception passed as the cause?

  ```java
  // Correct — original cause preserved in stack trace
  } catch (SQLException ex) {
      throw new DataAccessException("Failed to load user", ex);
  }

  // Wrong — original cause lost
  } catch (SQLException ex) {
      throw new DataAccessException("Failed to load user");
  }
  ```

- [ ] **`InterruptedException` handled correctly**: When catching
  `InterruptedException`, is `Thread.currentThread().interrupt()` called
  to restore the interrupted status before returning or rethrowing?

---

## Performance — Java

- [ ] **`StringBuilder` for string concatenation in loops**: Is the `+`
  operator used to concatenate strings inside a loop? Each `+` in a loop
  creates a new `String` object — use `StringBuilder.append()`.
- [ ] **`ArrayList` initial capacity**: Is a large `ArrayList` constructed
  without an initial capacity when the size is known in advance? Providing
  an initial capacity avoids repeated internal array resizing.
- [ ] **Unnecessary boxing/unboxing**: Are primitive wrappers (`Integer`,
  `Long`, `Double`) used in hot paths where primitives (`int`, `long`,
  `double`) would avoid autoboxing overhead?
- [ ] **`Stream` misuse**: Are `Stream` pipelines used for simple indexed
  or stateful iterations where a plain `for` loop is clearer and faster?
  Streams are not always preferable — choose based on readability and
  performance for the use case.
- [ ] **`HashMap` vs `EnumMap`**: Are `Enum` keys used in a `HashMap`
  instead of `EnumMap`? `EnumMap` is significantly faster for enum keys.

---

## Concurrency — Java

- [ ] **Shared mutable state protected**: Is mutable state accessed from
  multiple threads without synchronisation, `volatile`, or a concurrent
  collection?
- [ ] **`synchronized` scope minimal**: Is a `synchronized` block as narrow
  as possible, covering only the operations that require mutual exclusion?
  Over-synchronisation causes unnecessary contention.
- [ ] **`volatile` for visibility only**: Is `volatile` used as a substitute
  for atomicity? `volatile` guarantees visibility across threads but does
  not make compound operations atomic — use `AtomicInteger`, `AtomicReference`,
  or explicit locking for atomic updates.
- [ ] **Lock ordering consistent**: When multiple locks are acquired in
  sequence, is the order always the same across all code paths? Inconsistent
  lock ordering causes deadlocks.
- [ ] **`ThreadLocal` cleaned up**: Is `ThreadLocal.remove()` called when
  the value is no longer needed, particularly in thread-pool environments
  where threads are reused? Failure to remove causes memory leaks and
  data leakage between requests.

---

## Security — Java

The following items extend `security-review.md` with Java-specific
implementation guidance.

- [ ] **Parameterised JDBC queries**: Is `PreparedStatement` used for all
  queries that incorporate external input? String concatenation into a
  `Statement` is a SQL injection vulnerability.

  ```java
  // Correct
  PreparedStatement stmt = conn.prepareStatement(
      "SELECT * FROM users WHERE email = ?"
  );
  stmt.setString(1, email);

  // Wrong — SQL injection risk
  Statement stmt = conn.createStatement();
  stmt.executeQuery("SELECT * FROM users WHERE email = '" + email + "'");
  ```

- [ ] **Input validation at trust boundaries**: Is input from HTTP requests,
  message queues, or external APIs validated against expected types and
  ranges before processing? Use Bean Validation (`@NotNull`, `@Size`,
  `@Pattern`) on DTOs.
- [ ] **`SecureRandom` for security tokens**: Is `java.util.Random` used
  to generate tokens, passwords, or session identifiers?
  `java.security.SecureRandom` must be used for all security-sensitive
  random values.

  ```java
  // Correct
  SecureRandom random = new SecureRandom();
  byte[] token = new byte[32];
  random.nextBytes(token);

  // Wrong — not cryptographically secure
  Random random = new Random();
  ```

- [ ] **`XXE` prevention on XML parsing**: If XML is parsed, is the parser
  configured to disable external entity processing?

  ```java
  DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
  factory.setFeature(
      "http://xml.org/sax/features/external-general-entities", false
  );
  factory.setFeature(
      "http://xml.org/sax/features/external-parameter-entities", false
  );
  ```

- [ ] **Path traversal prevention**: Are file paths constructed from user
  input validated with `Path.normalize()` and checked to confirm they
  remain within the intended base directory?

  ```java
  Path base = Paths.get("/uploads").toRealPath();
  Path target = base.resolve(userInput).normalize();
  if (!target.startsWith(base)) {
      throw new SecurityException("Path traversal attempt detected");
  }
  ```

---

## Testing — Java

- [ ] **`@BeforeEach` over shared state**: Is test state initialised in
  instance fields without reset between tests? Use `@BeforeEach` to
  reinitialise state and prevent tests from affecting each other.
- [ ] **`assertThrows` for exceptions**: Is exception behaviour tested with
  a `try/catch` block instead of `assertThrows`? JUnit 5's `assertThrows`
  is more readable and verifies both the exception type and message.

  ```java
  // Correct
  ResourceNotFoundException ex = assertThrows(
      ResourceNotFoundException.class,
      () -> service.findById(999L)
  );
  assertThat(ex.getMessage()).contains("999");
  ```

- [ ] **Meaningful assertion messages**: Do `assertEquals` and `assertThat`
  calls include a descriptive message as the first argument for failures
  that would otherwise produce cryptic output?
- [ ] **No `Thread.sleep` in tests**: Is `Thread.sleep` used to wait for
  async operations in tests? Use `Awaitility` or `CompletableFuture.get()`
  with a timeout instead — `sleep` makes tests slow and flaky.