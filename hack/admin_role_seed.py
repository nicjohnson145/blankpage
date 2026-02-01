# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "requests>=2.32.5",
# ]
# ///
import requests


def must_ok(resp):
    if resp.status_code != 200:
        print(resp.text)
        raise ValueError(f"got non-200 status of {resp.status_code}")
    return resp


def main():
    base = "http://localhost:8080"
    pauth = f"{base}/pauth.v1beta1.PAuthService"
    roles = [
        "book-uploader",
        "book-updater",
        "shelf-admin",
        "view-only",
    ]

    # login
    resp = must_ok(
        requests.post(
            f"{pauth}/Login",
            json={
                "email": "admin@example.com",
                "password": "fake-admin-password",
            },
        )
    ).json()

    key = resp["access_key"]

    # get our user id
    resp = must_ok(
        requests.post(
            f"{pauth}/ReadUser",
            json={},
            headers={"Authorization": key},
        )
    ).json()
    user_id = resp["user"]["id"]

    # grant ourselves the roles
    for role in roles:
        must_ok(
            requests.post(
                f"{pauth}/GrantUserRole",
                json={
                    "user_id": user_id,
                    "role": role,
                },
                headers={"Authorization": key},
            )
        )


if __name__ == "__main__":
    main()
