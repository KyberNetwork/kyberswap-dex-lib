import unittest
import math
import liquidity_score_calc as liq

class TestGeometricMeanSafe(unittest.TestCase):
    def test_positive_numbers(self):
        data = [2, 8]
        self.assertAlmostEqual(liq.geometric_mean_safe(data), liq.geometric_mean(data))

        data = [1, 10, 100]
        self.assertAlmostEqual(liq.geometric_mean_safe(data), liq.geometric_mean(data))
    
    def test_with_zero(self):
        data = [0, 2, 8]
        self.assertEqual(liq.geometric_mean_safe(data), 0.0)
    
    def test_with_single_positive_number(self):
        data = [8]
        self.assertAlmostEqual(liq.geometric_mean_safe(data), liq.geometric_mean(data))


if __name__ == '__main__':
    unittest.main()