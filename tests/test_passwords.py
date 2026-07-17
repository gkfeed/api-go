import pytest

from api.passwords import InvalidPasswordHashError, hash_password, verify_password


def test_argon2id_hashes_use_unique_salts() -> None:
    first = hash_password("correct horse battery staple")
    second = hash_password("correct horse battery staple")

    assert first != second
    assert first.startswith("$argon2id$v=19$m=19456,t=2,p=1$")
    assert verify_password(first, "correct horse battery staple")
    assert not verify_password(first, "wrong")


def test_plaintext_is_not_a_valid_password_hash() -> None:
    with pytest.raises(InvalidPasswordHashError):
        verify_password("plaintext", "plaintext")


def test_excessive_argon2_parameters_are_rejected_before_verification() -> None:
    encoded_hash = hash_password("password").replace("m=19456", "m=1048577")

    with pytest.raises(InvalidPasswordHashError):
        verify_password(encoded_hash, "password")
