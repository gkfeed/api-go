import sys

from api.passwords import hash_password


def main() -> None:
    password = sys.stdin.read(1 << 20).removesuffix("\n").removesuffix("\r")
    if not password:
        raise SystemExit("read password from standard input")
    print(hash_password(password))


if __name__ == "__main__":
    main()
