# Python — Language-Specific Review Checklist

Load this file after `review-checklist.md` and `security-review.md` when
reviewing Python code. This file contains all Python-specific checks that
were extracted from those common files, ensuring no coverage is lost for
Python projects.

---

## Logic & Correctness — Python

- [ ] **Comprehensions**: Are list/dict/set comprehensions readable?
  More than two levels of nesting is a refactoring signal — extract into
  a named function or a standard loop.
- [ ] **Data structures**: Is `set` used for membership checks and uniqueness
  enforcement instead of `list`? Is `dict` used appropriately vs a custom
  class?
- [ ] **Truthiness traps**: Is `== None` used where `is None` is correct?
  Is `len(x) == 0` used where `not x` is idiomatic?
- [ ] **Mutable default arguments**: Does any function use a mutable object
  as a default argument (`def f(x=[])` or `def f(d={})`)?
  This is a classic silent bug — the default is shared across all calls.
- [ ] **Generator exhaustion**: Is a generator being iterated more than once?
  Generators are exhausted after the first iteration and silently produce
  nothing on subsequent calls.

---

## Architecture & Design — Python

- [ ] **`__init__.py` exports**: Does the module's public API match what is
  exported in `__init__.py`? Internal helpers should not be re-exported.
- [ ] **Dataclasses vs dicts**: Is a plain `dict` used where a `dataclass`
  or `TypedDict` would make the structure explicit and safer?
- [ ] **Class vs function**: Is a class being used purely as a namespace
  for static methods? A module with functions is idiomatic Python and
  preferable in that case.

---

## Readability & Maintainability — Python

- [ ] **Type hints**: Are type hints present on all public function signatures
  (parameters and return type)? This is required per project convention.
- [ ] **Docstrings**: Are docstrings present on all public functions, classes,
  and modules? Docstrings must follow Google Style format:

  ```python
  def process(data: list[str]) -> dict[str, int]:
      """Counts occurrences of each string in the input list.

      Args:
          data: A list of strings to process.

      Returns:
          A dictionary mapping each unique string to its count.

      Raises:
          ValueError: If data is None.
      """
  ```

- [ ] **f-strings vs format**: Are f-strings used for readability where
  appropriate, rather than `%` formatting or `.format()`?

---

## Error Handling — Python

- [ ] **Exception specificity**: Is `except Exception` or bare `except` used?
  These catch everything including `KeyboardInterrupt` and `SystemExit`.
  Always catch the most specific exception type available.
- [ ] **Re-raise with context**: When re-raising, is `raise NewError() from exc`
  used to preserve the original traceback?

  ```python
  # Correct — preserves original context
  except HTTPError as exc:
      raise ExternalServiceError("Payment gateway unavailable") from exc

  # Wrong — loses original traceback
  except HTTPError:
      raise ExternalServiceError("Payment gateway unavailable")
  ```

- [ ] **Context managers**: Are files, database connections, locks, and other
  resources always opened with `with` statements to guarantee cleanup on
  both success and exception paths?

  ```python
  # Correct
  with open(path, "r") as f:
      content = f.read()

  # Wrong — file not closed on exception
  f = open(path, "r")
  content = f.read()
  f.close()
  ```

---

## Performance — Python

- [ ] **Async correctness**: Is `await` used on every coroutine call?
  Is any synchronous blocking I/O (file reads, `time.sleep`, synchronous
  HTTP calls) running inside an `async def` function without being
  delegated to a thread pool (`run_in_executor`)?
- [ ] **`with` for resources**: Are database sessions, HTTP clients, and
  file handles always closed via `with` or `async with`?
- [ ] **Unnecessary list materialisation**: Is `list()` called on a generator
  or iterator when only iteration is needed? Keep it lazy where possible.

---

## Security — Python

The following items extend `security-review.md` with Python-specific
implementation guidance.

- [ ] **Input validation**: Is user and external API input validated using
  Pydantic models (or equivalent schema validation library) before processing?
  Do not validate manually with `if/else` chains on raw dicts.

  ```python
  # Correct
  class CreateOrderRequest(BaseModel):
      user_id: int
      amount: Decimal = Field(gt=0)

  # Wrong — manual validation is error-prone and incomplete
  def create_order(data: dict):
      if "user_id" not in data:
          raise ValueError("missing user_id")
  ```

- [ ] **Cryptographically secure random**: Is the `secrets` module used for
  generating tokens, passwords, and security-sensitive identifiers?
  The `random` module is **not** cryptographically secure and must never
  be used for security purposes.

  ```python
  # Correct
  import secrets
  token = secrets.token_urlsafe(32)

  # Wrong — random is not cryptographically secure
  import random
  import string
  token = ''.join(random.choices(string.ascii_letters, k=32))
  ```

- [ ] **SQL injection via ORM raw queries**: Even with an ORM, check for
  `session.execute(text(f"... {user_input}"))` or similar raw string
  formatting in query builders.
- [ ] **`subprocess` safety**: Are `subprocess` calls using a list of
  arguments (`subprocess.run(["cmd", arg])`) rather than `shell=True`
  with user-controlled input?
- [ ] **`pickle` on untrusted data**: Is `pickle.loads` called on any data
  that originates from user input or an external source?
  Pickle deserialization of untrusted data allows arbitrary code execution.
- [ ] **`eval` / `exec`**: Is `eval()` or `exec()` called on any
  user-supplied content? These must never be used with untrusted input.