import base64
import binascii
import re

from argon2.exceptions import InvalidHashError, VerificationError, VerifyMismatchError
from argon2.low_level import Type, hash_secret, verify_secret

ARGON2_MEMORY = 19_456
ARGON2_ITERATIONS = 2
ARGON2_PARALLELISM = 1
ARGON2_HASH_LENGTH = 32
ARGON2_SALT_LENGTH = 16
DUMMY_PASSWORD_HASH = (
    "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$"
    "2Jg2r6u8F0x2QOeqRclG9ji4W1eAV0cYh8E1Fus2Yfk"
)
HASH_PATTERN = re.compile(r"^\$argon2id\$v=19\$m=(\d+),t=(\d+),p=(\d+)\$([^$]+)\$([^$]+)$")


class InvalidPasswordHashError(ValueError):
    pass


def hash_password(password: str, *, salt: bytes | None = None) -> str:
    if salt is None:
        import secrets

        salt = secrets.token_bytes(ARGON2_SALT_LENGTH)
    return hash_secret(
        password.encode(),
        salt,
        time_cost=ARGON2_ITERATIONS,
        memory_cost=ARGON2_MEMORY,
        parallelism=ARGON2_PARALLELISM,
        hash_len=ARGON2_HASH_LENGTH,
        type=Type.ID,
    ).decode()


def verify_password(encoded_hash: str, password: str) -> bool:
    _validate_hash(encoded_hash)
    try:
        return verify_secret(encoded_hash.encode(), password.encode(), Type.ID)
    except VerifyMismatchError:
        return False
    except (InvalidHashError, VerificationError) as error:
        raise InvalidPasswordHashError("invalid Argon2id password hash") from error


def _validate_hash(encoded_hash: str) -> None:
    match = HASH_PATTERN.fullmatch(encoded_hash)
    if match is None:
        raise InvalidPasswordHashError("invalid Argon2id password hash")
    memory, iterations, parallelism = (int(value) for value in match.group(1, 2, 3))
    if (
        not 8 * parallelism <= memory <= 1_048_576
        or not 1 <= iterations <= 20
        or not 1 <= parallelism <= 16
    ):
        raise InvalidPasswordHashError("invalid Argon2id password hash")
    try:
        salt = _decode_unpadded_base64(match.group(4))
        digest = _decode_unpadded_base64(match.group(5))
    except (binascii.Error, ValueError) as error:
        raise InvalidPasswordHashError("invalid Argon2id password hash") from error
    if not 16 <= len(salt) <= 64 or not 16 <= len(digest) <= 64:
        raise InvalidPasswordHashError("invalid Argon2id password hash")


def _decode_unpadded_base64(value: str) -> bytes:
    return base64.b64decode(value + "=" * (-len(value) % 4), validate=True)
