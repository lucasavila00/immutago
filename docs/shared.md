# Shared objects

Use `im.Shared` for data that several parts of a program need to keep and read,
such as a product description. Functions can return pointers to it, and other
Shared objects can store those pointers. The checker rejects changes to its fields.

This order keeps a pointer to the product. Reading the order does not copy the
product:

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

Choose [Scoped](scoped.md) instead when a function must not keep a pointer after
it returns.

## Allowed forms

The [common forms](common-forms.md) and the table below give the complete list
of allowed operations. Other protected operations are rejected.

`*Product` also permits these forms:

| Allowed form | Example |
| --- | --- |
| Store in a marked field | `order := Order{Product: p}` |
| Store in a local slice or map before Freeze | `values := []*Product{p}` |
| Store in a package variable | `saved = p` |
| Read a stored pointer | `p := order.Product`, `p := values[i]` |
| Return or receive a pointer from a named function | `return p`, `p := product()` |

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `p.Price = 9` | `p = &Product{Price: 9}` |
| `value := Product{}; p := &value` | `value := Product{}; p := new(value)` |
| `var value any = p` | Keep a concrete `*Product` variable. |
