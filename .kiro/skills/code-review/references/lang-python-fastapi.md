# Python + FastAPI — Framework-Specific Review Checklist

Load this file after `lang-python.md` when reviewing code that uses FastAPI.
This file covers patterns and pitfalls specific to FastAPI — it does not
repeat items already in `review-checklist.md` or `lang-python.md`.

---

## Routing & Endpoints

- [ ] **Response model declared**: Does every endpoint declare a
  `response_model`? Without it, FastAPI cannot validate or filter the
  response, and internal fields may be accidentally exposed.

  ```python
  # Correct
  @router.get("/users/{user_id}", response_model=UserResponse)
  async def get_user(user_id: int): ...

  # Wrong — no response filtering or validation
  @router.get("/users/{user_id}")
  async def get_user(user_id: int): ...
  ```

- [ ] **Status codes explicit**: Are non-200 success responses declaring
  `status_code` explicitly (e.g., `status_code=201` for creation endpoints)?
- [ ] **Path parameter types**: Are path parameters typed (`user_id: int`
  not `user_id: str`) so FastAPI validates and converts them automatically?
- [ ] **HTTP method correctness**: Are `GET` endpoints free of side effects?
  Are mutations using `POST`, `PUT`, `PATCH`, or `DELETE` as appropriate?

---

## Dependency Injection (`Depends`)

- [ ] **Auth dependency on every protected route**: Is the authentication
  or authorization dependency applied to every route that requires it?
  It is easy to forget `Depends(get_current_user)` on a new endpoint.
- [ ] **No business logic in dependencies**: Do `Depends` functions handle
  only cross-cutting concerns (auth, DB session, rate limiting)?
  Business logic belongs in the service layer, not in a dependency.
- [ ] **Database session lifecycle**: Is the DB session created via a
  `Depends` generator that yields and closes in a `finally` block,
  ensuring the session is always released?

  ```python
  # Correct
  def get_db():
      db = SessionLocal()
      try:
          yield db
      finally:
          db.close()
  ```

---

## Request & Response Models (Pydantic)

- [ ] **Separate request and response models**: Is a single model used for
  both input and output? Input models should not expose fields like `id`,
  `created_at`, or `password_hash` that are set server-side.
- [ ] **Field validation constraints**: Are `Field(gt=0)`, `Field(max_length=...)`,
  and similar constraints used to enforce business rules at the schema level,
  rather than manually in the service layer?
- [ ] **`model_config` with `from_attributes`**: If a response model is
  populated from an ORM object, is `model_config = ConfigDict(from_attributes=True)`
  set? Missing this causes silent serialization failures.
- [ ] **No mutable defaults in Pydantic models**: Is `default_factory` used
  for mutable defaults (lists, dicts) rather than a direct default value?

  ```python
  # Correct
  tags: list[str] = Field(default_factory=list)

  # Wrong — shared mutable default
  tags: list[str] = []
  ```

---

## Async & Performance

- [ ] **`async def` for I/O-bound endpoints**: Are endpoints that perform
  database queries or external HTTP calls declared `async def`?
  Synchronous `def` endpoints run in a thread pool — acceptable but not
  optimal for I/O-bound work.
- [ ] **No blocking calls in async endpoints**: Are synchronous blocking
  operations (e.g., `time.sleep`, synchronous ORM calls in an async context)
  absent from `async def` endpoints? They block the event loop.
- [ ] **Background tasks for fire-and-forget**: Are side effects that do not
  affect the response (e.g., sending an email, writing an audit log) handled
  via `BackgroundTasks` rather than awaited in the request handler?

---

## Error Handling

- [ ] **`HTTPException` with meaningful detail**: Are `HTTPException` instances
  raised with a `detail` message that is helpful to the API consumer without
  leaking internals?
- [ ] **Global exception handlers**: Is there a global exception handler
  (`@app.exception_handler`) for unexpected errors that returns a
  structured error response instead of a raw 500?
- [ ] **Validation error format**: Is `RequestValidationError` handled to
  return a consistent error format matching the rest of the API, rather than
  FastAPI's default validation error structure (if the API has a custom format)?

---

## Security

- [ ] **CORS configuration**: Is `CORSMiddleware` configured with an explicit
  `allow_origins` list? `allow_origins=["*"]` is acceptable only for fully
  public APIs with no authentication.
- [ ] **No secrets in query parameters**: Are tokens or API keys passed in
  request headers or the body — not as query parameters that appear in
  server logs?
- [ ] **Rate limiting on sensitive endpoints**: Are authentication endpoints
  (`/login`, `/token`, `/reset-password`) protected against brute force
  with rate limiting middleware?

---

## Testing

- [ ] **`TestClient` or `AsyncClient` used**: Are endpoint tests using
  FastAPI's `TestClient` (sync) or `httpx.AsyncClient` with
  `transport=ASGITransport(app=app)` (async) rather than testing service
  functions directly?
- [ ] **Dependencies overridden in tests**: Are auth and database dependencies
  overridden via `app.dependency_overrides` in tests, rather than requiring
  a live database or real credentials?
- [ ] **Response schema validated in tests**: Do tests assert on the
  response body structure, not just the status code?