# Workflow Example: Cross-Service Orchestration

An e-commerce scenario with three services (Inventory, Orders, Notifications) that demonstrates how AI agents orchestrate multi-step workflows across services — no glue code, no workflow engine.

## The Workflow

When a user says _"Order a ThinkPad for alice and send her a confirmation"_, the agent figures out the steps:

```
1. InventoryService.Search     → Find the product
2. InventoryService.CheckStock → Verify availability
3. InventoryService.ReserveStock → Decrement inventory
4. OrderService.PlaceOrder     → Create the order
5. NotificationService.Send    → Email confirmation
```

No code connects these steps — the agent reads the tool descriptions and chains the calls itself.

## Run

```bash
go run .
```

## Services

| Service | Tools | Purpose |
|---------|-------|---------|
| InventoryService | Search, CheckStock, ReserveStock | Product catalog and stock management |
| OrderService | PlaceOrder, GetOrder, ListOrders | Order creation and lookup |
| NotificationService | Send, List | Email/SMS/Slack notifications |

## Example Prompts

Try these with Claude Code (`micro mcp serve`) or any MCP-compatible agent:

- "What laptops do you have in stock?"
- "Order a ThinkPad for alice@example.com and send her a confirmation"
- "Check if 'The Go Programming Language' is available" (it's out of stock!)
- "Order 3 Go Gopher t-shirts for bob@example.com, reserve the stock, and notify him via Slack"
- "Show me all orders and notifications for alice"

## Why This Matters

Traditional approach:
```go
// 50+ lines of glue code wiring services together
func handleOrder(req OrderRequest) {
    product, err := inventoryClient.CheckStock(req.SKU)
    if err != nil { ... }
    if product.InStock < req.Quantity { ... }
    _, err = inventoryClient.ReserveStock(req.SKU, req.Quantity)
    if err != nil { ... }
    order, err := orderClient.PlaceOrder(...)
    if err != nil { ... }
    _, err = notificationClient.Send(...)
    // ...
}
```

Agent approach:
```
User: "Order a ThinkPad for alice and confirm via email"
Agent: [reads tool descriptions, chains 5 calls, handles the out-of-stock case]
```

The agent handles the orchestration. You just write the individual services with good documentation.
