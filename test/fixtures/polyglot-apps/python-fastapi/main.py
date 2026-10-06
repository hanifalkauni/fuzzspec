from fastapi import FastAPI, HTTPException, Query

app = FastAPI(title="Python FastAPI Sample API", version="1.0.0")

@app.get("/users/{user_id}")
def get_user(user_id: int):
    # Bug: Unhandled division by zero or negative lookup panic
    if user_id < 0:
        raise ValueError(f"Traceback (most recent call last):\n  File 'app/main.py', line 9, in get_user\n    lookup_user(user_id)\nValueError: Negative user id not allowed: {user_id}")
    return {"user_id": user_id, "name": f"User_{user_id}"}
