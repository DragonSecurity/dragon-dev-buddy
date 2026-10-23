---
description: "Reviewing one handler for security is code review, not the pre-deploy gate."
tags: [routing]
max_turns: 12
timeout_seconds: 240
allowed_tools: [Read, Glob, Grep, Skill]
---

Review this login handler for security problems before I merge it:

```python
@app.post('/login')
def login():
    user = db.execute(f"SELECT * FROM users WHERE email = '{request.form['email']}'").fetchone()
    if user and user['password'] == request.form['password']:
        session['uid'] = user['id']
        return redirect(request.args.get('next', '/'))
    return 'bad login', 401
```
