---
description: "A wrong result with the code attached is a bug to reproduce and fix."
tags: [routing]
max_turns: 12
timeout_seconds: 240
allowed_tools: [Read, Glob, Grep, Skill]
---

Our invoice total comes out as NaN whenever a customer has zero line items. Why? Here's the function:

```js
function total(items) {
  return items.reduce((sum, i) => sum + i.price * i.qty) / items.length * items.length;
}
```
