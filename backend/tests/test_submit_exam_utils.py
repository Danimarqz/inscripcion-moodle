import unittest

from services.exam.submit_exam import normalize_dni, validate_answer_option, validate_dni_nie


class TestSubmitExamUtils(unittest.TestCase):
    def test_normalize_dni_trims_whitespace_and_uppercases(self):
        self.assertEqual(normalize_dni("  ab123456c "), "AB123456C")

    def test_validate_dni_accepts_valid_spanish_dni(self):
        self.assertTrue(validate_dni_nie("12345678Z"))

    def test_validate_dni_accepts_valid_nie(self):
        self.assertTrue(validate_dni_nie("x1234567l"))

    def test_validate_dni_rejects_invalid_letter(self):
        self.assertFalse(validate_dni_nie("12345678A"))

    def test_validate_dni_rejects_invalid_length(self):
        self.assertFalse(validate_dni_nie("1234"))

    def test_validate_answer_option_returns_upper(self):
        self.assertEqual(validate_answer_option("b"), "B")

    def test_validate_answer_option_raises_for_invalid(self):
        with self.assertRaises(Exception):
            validate_answer_option("E")


if __name__ == "__main__":
    unittest.main()
