import numpy as np
from scipy.stats import entropy

# Step 1: Read the data from the text file and parse it, output is min score
def get_top_pools(pool_scores: list, field: str, target_factor: float):
        
    # Convert the data into a structured NumPy array for easier processing
    score_values = np.array([entry[field] for entry in pool_scores])

    log_score_values = np.log10(score_values + 1)  # Adding 1 to avoid log(0) issue
    # Apply the function to each metric
    selected_score_values = entropy_based_selection(log_score_values, target_factor)
    min_score = min(10**selected_score_values - 1)

    print(f"Entropy selected result for {field} mean with target_factor_entropy {target_factor}:")
    print(f"Length of total values: {len(log_score_values)}, with min value {min(10**log_score_values - 1)}")
    print(f"Length selected values: {len(selected_score_values)} with min value {min_score}")
    print(f"Length {field} values greater than {min_score}: {np.sum(score_values >= min_score)}")

    return min_score


# Function to perform the entropy-based selection for a given metric
def entropy_based_selection(log_values, target_factor: float):
    # Step 1: Calculate initial entropy of the entire array
    sum_log_values = np.sum(log_values)
    if sum_log_values == 0:
        raise ValueError("The sum of log-scaled values is zero, which causes invalid probabilities.")

    probabilities_log = log_values / sum_log_values  # Normalize values to probabilities

    # Ensure probabilities are non-negative
    if np.any(probabilities_log < 0):
        raise ValueError("Negative probabilities encountered.")

    initial_entropy_log = entropy(probabilities_log)  # Calculate entropy of the original array

    if np.isinf(initial_entropy_log) or np.isnan(initial_entropy_log):
        raise ValueError("Initial entropy calculation resulted in invalid value (inf or NaN).")

    # Step 2: Get unique values and counts, sort in ascending order
    unique_values, counts = np.unique(log_values, return_counts=True)
    sorted_indices = np.argsort(unique_values)
    unique_values_sorted = unique_values[sorted_indices]
    counts_sorted = counts[sorted_indices]

    # Step 3: Initialize variables
    entropy_log_top_values = log_values.copy()
    cumulative_sum_log = sum_log_values

    # Step 4: Remove groups of equal values
    for value, count in zip(unique_values_sorted, counts_sorted):
        # Remove all instances of the current smallest unique value
        mask = entropy_log_top_values != value
        cumulative_sum_log -= value * count
        entropy_log_top_values = entropy_log_top_values[mask]

        if cumulative_sum_log == 0:
            break  # Avoid dividing by zero

        # Recalculate probabilities and entropy with the remaining values
        probabilities_subset = entropy_log_top_values / cumulative_sum_log
        current_entropy_log = entropy(probabilities_subset)

        # Step 5: Break the loop once the entropy is reduced by the target factor
        if current_entropy_log <= target_factor * initial_entropy_log:
            break

    # Step 6: Output the final selected values
    return np.sort(entropy_log_top_values)
