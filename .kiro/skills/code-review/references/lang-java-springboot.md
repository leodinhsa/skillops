# Java + Spring Boot — Framework-Specific Review Checklist

Load this file after `lang-java.md` when reviewing code that uses Spring Boot.
This file covers patterns and pitfalls specific to Spring Boot — it does not
repeat items already in `review-checklist.md` or `lang-java.md`.

---

## Dependency Injection & Bean Management

- [ ] **Constructor injection preferred**: Is `@Autowired` on fields used
  instead of constructor injection? Field injection hides dependencies,
  makes testing harder, and prevents immutability.

  ```java
  // Correct — constructor injection
  @Service
  @RequiredArgsConstructor
  public class OrderService {
      private final OrderRepository orderRepository;
      private final PaymentClient paymentClient;
  }

  // Wrong — field injection
  @Service
  public class OrderService {
      @Autowired
      private OrderRepository orderRepository;
  }
  ```

- [ ] **Bean scope appropriate**: Is `@Scope("prototype")` or
  `@RequestScope` used where a new instance per request is required?
  Singleton beans (the default) must not hold request-scoped or
  mutable per-user state.
- [ ] **Circular dependency absent**: Does the context load without
  `@Lazy` workarounds that mask a design problem? Circular dependencies
  indicate a layering violation that should be resolved structurally.
- [ ] **No `ApplicationContext.getBean()`**: Is `getBean()` used to
  manually pull beans from the context at runtime? This bypasses DI
  and is a service-locator anti-pattern.

---

## `@Transactional`

- [ ] **Correct layer**: Is `@Transactional` placed on the service layer,
  not on the repository or controller layer?
- [ ] **Public methods only**: Is `@Transactional` applied to non-public
  methods? Spring's proxy mechanism cannot intercept non-public methods —
  the annotation is silently ignored.
- [ ] **Self-invocation trap**: Is a `@Transactional` method called from
  another method in the same class? Self-invocation bypasses the proxy
  and the transaction is not applied.
- [ ] **Propagation understood**: Is a non-default propagation level
  (`REQUIRES_NEW`, `NESTED`, etc.) used? Verify that the behaviour under
  both commit and rollback is intentional and documented.
- [ ] **Read-only optimisation**: Are `@Transactional(readOnly = true)`
  annotated on query-only service methods to allow the persistence
  provider to apply read optimisations?
- [ ] **Exception rollback rules**: Does the transaction roll back on
  the correct exception types? By default, Spring only rolls back on
  `RuntimeException` and `Error` — checked exceptions do not trigger
  rollback unless configured explicitly.

  ```java
  // Explicit rollback for checked exception
  @Transactional(rollbackFor = InsufficientFundsException.class)
  public void transfer(long amount) throws InsufficientFundsException { ... }
  ```

---

## REST Controllers

- [ ] **`@RestController` used**: Is `@Controller` + `@ResponseBody`
  on every method used instead of `@RestController`?
- [ ] **Thin controllers**: Does the controller contain business logic,
  validation beyond basic constraint annotations, or direct repository
  calls? Controllers must delegate to the service layer immediately.
- [ ] **`@Valid` on request bodies**: Is `@Valid` (or `@Validated`)
  present on `@RequestBody` parameters to trigger Bean Validation?
  Without it, constraint annotations on the DTO are never evaluated.

  ```java
  // Correct
  @PostMapping("/orders")
  public ResponseEntity<OrderResponse> create(
          @Valid @RequestBody CreateOrderRequest request) { ... }

  // Wrong — @Valid missing, constraints ignored
  @PostMapping("/orders")
  public ResponseEntity<OrderResponse> create(
          @RequestBody CreateOrderRequest request) { ... }
  ```

- [ ] **Consistent response format**: Does the endpoint return a response
  wrapped in the project's standard `ApiResponse<T>` (or equivalent)
  rather than raw objects or inconsistent structures?
- [ ] **HTTP status codes correct**: Does creation return `201 Created`?
  Does a void operation return `204 No Content`? Does a not-found case
  return `404` rather than `200` with a null body?

---

## Exception Handling

- [ ] **`@ControllerAdvice` used**: Are exceptions handled in a centralised
  `@ControllerAdvice` class rather than with `try/catch` in each
  controller method?
- [ ] **Custom exception hierarchy**: Does the project define its own
  exception classes (e.g., `ResourceNotFoundException`,
  `BusinessRuleViolationException`) rather than throwing generic
  `RuntimeException` or Spring framework exceptions directly?
- [ ] **No stack traces in responses**: Does the `@ControllerAdvice`
  produce a structured error response body without leaking the stack
  trace or internal class names to the client?

  ```java
  @ExceptionHandler(ResourceNotFoundException.class)
  @ResponseStatus(HttpStatus.NOT_FOUND)
  public ApiError handleNotFound(ResourceNotFoundException ex) {
      return new ApiError("NOT_FOUND", ex.getMessage());
      // Never include ex.getStackTrace() in the response
  }
  ```

---

## Data Access & JPA

- [ ] **No N+1 queries**: Are `@OneToMany` or `@ManyToMany` relationships
  fetched with `JOIN FETCH` in JPQL or `@EntityGraph` where the association
  is needed, rather than triggering lazy loading in a loop?
- [ ] **`FetchType.LAZY` as default**: Are associations using `EAGER`
  fetching? `EAGER` is a performance anti-pattern for most use cases —
  verify it is intentional and documented.
- [ ] **`Optional` on `findById`**: Does the code call `.get()` directly
  on the `Optional` returned by `findById()`? Use `.orElseThrow()` with
  a meaningful exception.

  ```java
  // Correct
  Order order = orderRepository.findById(id)
      .orElseThrow(() -> new ResourceNotFoundException("Order not found: " + id));

  // Wrong — throws NoSuchElementException with no context
  Order order = orderRepository.findById(id).get();
  ```

- [ ] **Parameterised queries**: Are JPQL or native queries using named
  parameters (`:param` or `?1`) rather than string concatenation?
- [ ] **`@Modifying` on update/delete queries**: Is `@Modifying` present
  on repository methods that execute UPDATE or DELETE JPQL statements?
  Without it, Spring Data throws an exception at runtime.

---

## Security (Spring Security)

- [ ] **`@PreAuthorize` on sensitive methods**: Are service methods that
  perform sensitive operations protected with `@PreAuthorize` or
  equivalent method security, not only at the controller level?
- [ ] **Password encoding**: Is `PasswordEncoder` (BCrypt recommended)
  used for all password storage and comparison? Plain-text comparison
  is never acceptable.
- [ ] **CSRF protection**: Is CSRF protection disabled only for stateless
  REST APIs using token-based auth? If the application uses session-based
  auth with a browser client, CSRF must be enabled.
- [ ] **Security context not stored in instance variables**: Is
  `SecurityContextHolder.getContext().getAuthentication()` called inside
  a method (not stored as a field) to avoid cross-request contamination
  in singleton beans?

---

## Configuration & Properties

- [ ] **No hardcoded configuration values**: Are environment-specific values
  (URLs, credentials, timeouts, feature flags) externalised to
  `application.properties` / `application.yml` or injected via
  `@Value` / `@ConfigurationProperties`?
- [ ] **`@ConfigurationProperties` for grouped config**: Is a group of
  related `@Value` annotations used instead of a `@ConfigurationProperties`
  class? The latter is type-safe, validatable, and easier to test.
- [ ] **Secrets not in properties files**: Are passwords, API keys, or
  tokens committed in `application.properties`? These must come from
  environment variables, Vault, AWS Secrets Manager, or equivalent.
- [ ] **Profile-specific configs separated**: Are `application-dev.yml`,
  `application-prod.yml` used to isolate environment-specific overrides
  rather than inline conditionals in application code?

---

## Async & Scheduling

- [ ] **`@Async` exception handling**: Does the `@Async` method return
  `Future` or `CompletableFuture` to allow callers to handle exceptions?
  `void` `@Async` methods swallow exceptions silently unless an
  `AsyncUncaughtExceptionHandler` is configured.
- [ ] **`@Scheduled` fixed-rate vs fixed-delay understood**: Is
  `fixedRate` used where `fixedDelay` is more appropriate (or vice versa)?
  `fixedRate` triggers on a wall-clock schedule regardless of execution
  duration; `fixedDelay` waits for the previous execution to complete.
- [ ] **Thread pool configured**: Are `@Async` and `@Scheduled` methods
  using a configured `ThreadPoolTaskExecutor`, not the default
  `SimpleAsyncTaskExecutor` which creates a new thread per invocation?

---

## Testing

- [ ] **`@WebMvcTest` for controller tests**: Are full `@SpringBootTest`
  contexts used for controller unit tests instead of `@WebMvcTest`?
  `@WebMvcTest` loads only the web layer and is significantly faster.
- [ ] **`@DataJpaTest` for repository tests**: Are full application contexts
  loaded for repository tests? `@DataJpaTest` spins up only the JPA layer
  with an in-memory database.
- [ ] **`MockMvc` assertions complete**: Do `MockMvc` tests assert on the
  response body content and `Content-Type` header, not only the HTTP
  status code?
- [ ] **`@MockBean` vs `@Mock`**: Is `@Mock` (Mockito) used inside a
  `@SpringBootTest` where `@MockBean` is required? `@Mock` does not
  replace the bean in the Spring context.