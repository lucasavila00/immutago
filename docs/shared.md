# Shared objects

`im.Shared` is an empty struct embedded as a marker for the checker.
Use it when callers need to keep pointers to read-only objects:

- Fields are read-only.
- Values can be copied, stored, and returned.
- Functions can return pointers to the object.
- Marked structs can store those pointers.

## Example: an order

The order stores a product pointer; reading it does not copy the product:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type Product struct {
    im.Shared
    Name string
    Price int
}

type Order struct {
    im.Shared
    Product *Product
    Quantity int
}

func product() *Product {
    return &Product{Name: "coffee", Price: 8}
}

func main() {
    order := Order{Product: product(), Quantity: 2}
    println(order.Product.Price * order.Quantity) // 16
    // order.Product.Price = 9 // Rejected: the product's fields are read-only.
}
```

Choose [ByValue](by-value.md) when callers can keep value copies but must not
store, return, or capture pointers to the object.

## Allowed forms — exhaustive

In addition to the [common forms](common-forms.md), `*Product` permits these forms:

| Allowed form | Example |
| --- | --- |
| Store in a marked field | `order := Order{Product: p}` |
| Store in a local slice or map before Freeze | `values := []*Product{p}` |
| Store in a package variable | `saved = p` |
| Read a stored pointer | `p := order.Product`, `p := values[i]` |
| Return or receive a pointer from a named function or allowed method | `return p`, `p := product()`, `p := order.ProductPointer()` |

## Rejected examples — non-exhaustive

| Rejected form | Allowed replacement |
| --- | --- |
| `p.Price = 9` | `p = &Product{Price: 9}` |
| `value := Product{}; p := &value` | `value := Product{}; p := new(value)` |
| `var value any = p` | Keep a concrete `*Product` variable. |
