# Python Feedback Examples

Concrete before/after examples illustrating the comment standards from
`feedback-template.md`, using Python syntax. Load this file when reviewing
Python code to calibrate tone, format, and level of specificity.

---

## P1-Blocker Example

**Scenario:** Missing ownership check allows any authenticated user to modify
another user's resource.

```
[P1-Blocker] Missing ownership check allows unauthorized data modification
Location: services/order_service.py:47

Current code:
  def update_order(self, order_id: int, data: UpdateOrderDTO) -> Order:
      order = self.repo.get_by_id(order_id)
      return self.repo.update(order, data)

Problem:
  Any authenticated user can update any order by guessing its ID.
  There is no check that the requesting user owns this order.

Suggested fix:
  def update_order(
      self, order_id: int, data: UpdateOrderDTO, requesting_user_id: int
  ) -> Order:
      order = self.repo.get_by_id(order_id)
      if order.user_id != requesting_user_id:
          raise ForbiddenError("You do not have permission to modify this order.")
      return self.repo.update(order, data)

Reason:
  Ownership must be verified in the service layer before any mutation,
  not assumed by the caller.
```

---

## P2-Should Fix Example

**Scenario:** N+1 query inside a loop degrades performance under load.

```
[P2-Should Fix] N+1 query — database called inside loop
Location: api/routers/reports.py:83

Current code:
  def get_orders_with_items(user_id: int) -> list[OrderDTO]:
      orders = order_repo.get_by_user(user_id)
      for order in orders:
          order.items = item_repo.get_by_order(order.id)  # query per order
      return orders

Problem:
  Each iteration issues a separate database query. For a user with 100 orders,
  this produces 101 queries, causing significant latency under normal load.

Suggested fix:
  def get_orders_with_items(user_id: int) -> list[OrderDTO]:
      return order_repo.get_by_user_with_items(user_id)
      # repository uses joinedload or selectinload to fetch in a single query

Reason:
  Fetching related data eagerly in the repository layer eliminates the N+1
  pattern and keeps query logic out of the service layer.
```

---

## P3-Suggestion Example

**Scenario:** Catching a broad exception loses diagnostic information.

```
[P3-Suggestion] Broad exception catch hides root cause in logs
Location: services/payment_service.py:112

Current code:
  try:
      response = payment_gateway.charge(amount, token)
  except Exception as exc:
      logger.error("Payment failed")
      raise PaymentError("Payment failed") from exc

Problem:
  Catching `Exception` broadly means all failure types — network timeout,
  invalid token, gateway rejection — produce identical log output.
  Debugging production incidents becomes unnecessarily difficult.

Suggested fix:
  try:
      response = payment_gateway.charge(amount, token)
  except GatewayTimeoutError as exc:
      logger.error("Payment gateway timeout: %s", exc)
      raise PaymentError("Service temporarily unavailable") from exc
  except GatewayRejectionError as exc:
      logger.warning("Payment rejected by gateway: %s", exc)
      raise PaymentDeclinedError("Payment was declined") from exc

Reason:
  Specific exception handling produces actionable log messages and allows
  callers to distinguish retriable errors from permanent failures.
```

---

## P4-Nit Example

**Scenario:** Variable name does not communicate intent.

```
[P4-Nit] Variable name does not communicate intent
Location: utils/date_helpers.py:19
Suggestion: Rename `d` to `expiry_date` — the current name requires
  reading the surrounding code to understand what it holds.
```

---

## Question Example

**Scenario:** Unclear whether a fallback is intentional or an oversight.

```
[Question] Is the empty-list fallback here intentional?
Location: services/recommendation_service.py:34

Current code:
  def get_recommendations(user_id: int) -> list[Product]:
      results = self.model.predict(user_id)
      return results or []

Question:
  Is returning an empty list the intended behaviour when the model returns
  None or an empty result — for example, for new users with no history?
  If so, a comment here would help future maintainers understand this is
  deliberate and not a missing error handler.
```

---

## Objective vs Subjective — Python Calibration

**Subjective (avoid):**
> "I don't like how you used a list comprehension here."

**Objective (use):**
```
[P4-Nit] Nested comprehension reduces readability
Location: data/transformers.py:56

Current code:
  result = [item.value for sublist in data for item in sublist if item.active]

Suggested fix:
  def _extract_active_values(data):
      for sublist in data:
          for item in sublist:
              if item.active:
                  yield item.value

  result = list(_extract_active_values(data))

Reason:
  Three levels of nesting in a comprehension exceed the readable threshold.
  A named generator makes the intent explicit and the logic individually testable.
```